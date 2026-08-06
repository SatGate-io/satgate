#!/usr/bin/env python3
"""Check live sitemap URLs for crawlable HTTP responses."""

from __future__ import annotations

import concurrent.futures
import sys
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET
from dataclasses import dataclass

SITEMAP_URL = "https://satgate.io/sitemap.xml"
TIMEOUT_SECONDS = 15
MAX_WORKERS = 12
OK_STATUSES = {200}


@dataclass(frozen=True)
class UrlCheck:
    url: str
    status: int | None
    final_url: str
    error: str = ""


def fetch_sitemap_urls() -> list[str]:
    with urllib.request.urlopen(SITEMAP_URL, timeout=TIMEOUT_SECONDS) as res:
        raw = res.read()
    root = ET.fromstring(raw)
    ns = {"sm": "http://www.sitemaps.org/schemas/sitemap/0.9"}
    return sorted({el.text.strip() for el in root.findall(".//sm:loc", ns) if el.text})


def request_url(url: str, method: str) -> UrlCheck:
    req = urllib.request.Request(url, method=method, headers={"User-Agent": "SatGate sitemap health check"})
    with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as res:
        return UrlCheck(url=url, status=res.status, final_url=res.geturl())


def check_url(url: str) -> UrlCheck:
    try:
        return request_url(url, "HEAD")
    except urllib.error.HTTPError as exc:
        if exc.code in {403, 405}:
            try:
                return request_url(url, "GET")
            except Exception as retry_exc:  # noqa: BLE001
                return UrlCheck(url=url, status=None, final_url=url, error=str(retry_exc)[:200])
        return UrlCheck(url=url, status=exc.code, final_url=exc.geturl(), error=str(exc)[:200])
    except Exception as exc:  # noqa: BLE001
        return UrlCheck(url=url, status=None, final_url=url, error=str(exc)[:200])


def main() -> None:
    urls = fetch_sitemap_urls()
    with concurrent.futures.ThreadPoolExecutor(max_workers=MAX_WORKERS) as pool:
        checks = list(pool.map(check_url, urls))

    bad = [check for check in checks if check.status not in OK_STATUSES or check.final_url != check.url]
    print(f"live sitemap urls scanned: {len(checks)}")
    if bad:
        print("non-200 or redirected sitemap URLs:")
        for check in bad:
            detail = f"{check.url} -> status={check.status or 'ERROR'}"
            if check.final_url != check.url:
                detail += f" final={check.final_url}"
            if check.error:
                detail += f" error={check.error}"
            print(detail)
        raise SystemExit(1)
    print("all live sitemap URLs returned 200 without redirects")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(130)
