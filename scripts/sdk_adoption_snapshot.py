#!/usr/bin/env python3
"""Collect a public SatGate SDK adoption snapshot.

This intentionally uses public/package-manager signals first. It does not add SDK
phone-home telemetry and does not inspect developer payloads.

Outputs markdown by default so it can be pasted into weekly notes or dashboards.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import urllib.error
import urllib.request
from datetime import date, timedelta
from typing import Any

NPM_PACKAGE = "@satgate/sdk"
PYPI_PACKAGE = "satgate"
REPO = "SatGate-io/satgate"


def fetch_json(url: str) -> dict[str, Any]:
    request = urllib.request.Request(url, headers={"User-Agent": "SatGateAdoptionSnapshot/1.0"})
    with urllib.request.urlopen(request, timeout=20) as response:
        return json.loads(response.read().decode("utf-8"))


def maybe_fetch_json(url: str) -> tuple[dict[str, Any] | None, str | None]:
    try:
        return fetch_json(url), None
    except Exception as exc:  # noqa: BLE001 - ops snapshot should degrade gracefully
        return None, str(exc)


def gh_json(path: str) -> tuple[dict[str, Any] | None, str | None]:
    try:
        result = subprocess.run(
            ["gh", "api", path],
            check=True,
            capture_output=True,
            text=True,
            timeout=20,
        )
        return json.loads(result.stdout), None
    except Exception as exc:  # noqa: BLE001
        return None, str(exc)


def npm_downloads(period: str) -> tuple[int | None, str | None]:
    data, err = maybe_fetch_json(f"https://api.npmjs.org/downloads/point/{period}/{urllib.request.quote(NPM_PACKAGE, safe='')}")
    if not data:
        return None, err
    return int(data.get("downloads", 0)), None


def pypi_recent_downloads() -> tuple[dict[str, Any] | None, str | None]:
    # pypistats can lag; this endpoint returns recent day/week/month totals.
    return maybe_fetch_json(f"https://pypistats.org/api/packages/{PYPI_PACKAGE}/recent")


def main() -> int:
    today = date.today()
    week_ago = today - timedelta(days=6)

    npm_meta, npm_meta_err = maybe_fetch_json(f"https://registry.npmjs.org/{urllib.request.quote(NPM_PACKAGE, safe='')}")
    pypi_meta, pypi_meta_err = maybe_fetch_json(f"https://pypi.org/pypi/{PYPI_PACKAGE}/json")
    pypi_recent, pypi_recent_err = pypi_recent_downloads()
    npm_week, npm_week_err = npm_downloads("last-week")
    npm_day, npm_day_err = npm_downloads("last-day")
    npm_range, npm_range_err = npm_downloads(f"{week_ago.isoformat()}:{today.isoformat()}")
    gh_ready = bool(os.getenv("GITHUB_TOKEN"))
    if not gh_ready:
        try:
            subprocess.run(["gh", "auth", "status"], check=True, capture_output=True, text=True, timeout=10)
            gh_ready = True
        except Exception:  # noqa: BLE001
            gh_ready = False
    gh_views, gh_views_err = gh_json(f"repos/{REPO}/traffic/views") if gh_ready else (None, "Set GITHUB_TOKEN or run gh auth login for GitHub traffic")
    gh_clones, gh_clones_err = gh_json(f"repos/{REPO}/traffic/clones") if gh_ready else (None, "Set GITHUB_TOKEN or run gh auth login for GitHub traffic")

    print(f"# SatGate SDK Adoption Snapshot — {today.isoformat()}\n")
    print("## Package versions")
    print(f"- npm `{NPM_PACKAGE}`: `{(npm_meta or {}).get('dist-tags', {}).get('latest', 'unknown')}`" + (f" _(error: {npm_meta_err})_" if npm_meta_err else ""))
    print(f"- PyPI `{PYPI_PACKAGE}`: `{(pypi_meta or {}).get('info', {}).get('version', 'unknown')}`" + (f" _(error: {pypi_meta_err})_" if pypi_meta_err else ""))

    print("\n## Public download signals")
    print(f"- npm last day: `{npm_day if npm_day is not None else 'unknown'}`" + (f" _(error: {npm_day_err})_" if npm_day_err else ""))
    print(f"- npm last week: `{npm_week if npm_week is not None else 'unknown'}`" + (f" _(error: {npm_week_err})_" if npm_week_err else ""))
    print(f"- npm explicit 7-day range: `{npm_range if npm_range is not None else 'unknown'}`" + (f" _(error: {npm_range_err})_" if npm_range_err else ""))
    if pypi_recent and "data" in pypi_recent:
        data = pypi_recent["data"]
        print(f"- PyPI last day: `{data.get('last_day', 'unknown')}`")
        print(f"- PyPI last week: `{data.get('last_week', 'unknown')}`")
        print(f"- PyPI last month: `{data.get('last_month', 'unknown')}`")
    else:
        print(f"- PyPI recent downloads: `unknown`" + (f" _(error: {pypi_recent_err})_" if pypi_recent_err else ""))

    print("\n## GitHub traffic signals")
    if gh_views:
        print(f"- repo views 14d: `{gh_views.get('count', 'unknown')}` total / `{gh_views.get('uniques', 'unknown')}` unique")
    else:
        print(f"- repo views 14d: `unknown` _(error: {gh_views_err})_")
    if gh_clones:
        print(f"- repo clones 14d: `{gh_clones.get('count', 'unknown')}` total / `{gh_clones.get('uniques', 'unknown')}` unique")
    else:
        print(f"- repo clones 14d: `unknown` _(error: {gh_clones_err})_")

    print("\n## Interpretation notes")
    print("- npm/PyPI download counts include CI, mirrors, bots, and retries. Watch slope, not absolute truth.")
    print("- The high-intent signal is docs/request-access conversion after `SatGateAuthError`, not raw downloads.")
    print("- Do not add SDK phone-home telemetry unless it is explicit, documented, payload-free, and opt-out.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
