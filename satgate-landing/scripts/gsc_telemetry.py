#!/usr/bin/env python3
"""SatGate SEO telemetry from Google Search Console.

Generates a Markdown report and JSON artifact covering:
- page/query performance
- cluster rollups
- CTR and position opportunities
- selected URL Inspection index states

Requires a Search Console OAuth token with webmasters scope at:
  /Users/mattdean/.gbrain/google-search-console-tokens.json
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import pathlib
import re
import sys
import urllib.request
import xml.etree.ElementTree as ET
from collections import defaultdict
from typing import Any

from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from googleapiclient.discovery import build

SITE = "sc-domain:satgate.io"
BASE_URL = "https://satgate.io"
SITEMAP_URL = "https://satgate.io/sitemap.xml"
TOKEN_PATH = pathlib.Path("/Users/mattdean/.gbrain/google-search-console-tokens.json")
DEFAULT_OUT_DIR = pathlib.Path("/Users/mattdean/Obsidian/AgentMemory/Agent-OpenClaw/reports")
DEFAULT_JSON_DIR = pathlib.Path("/Users/mattdean/.openclaw/workspace/reports")
SCOPES = [
    "https://www.googleapis.com/auth/webmasters",
    "openid",
    "https://www.googleapis.com/auth/userinfo.email",
]

CLUSTERS: list[tuple[str, list[str]]] = [
    ("Tools/Templates", ["/tools", "calculator", "generator", "template", "assessment", "grader", "/data/"]),
    ("MCP", ["/mcp", "mcp-"]),
    ("Comparisons", ["/compare"]),
    ("Integrations", ["/integrations", "/satgate-for-"]),
    ("AI Agent Spend", ["/ai-agent", "/agent-spending", "/ai-api-budget", "/runaway"]),
    ("Economic Firewall", ["/economic-firewall", "what-is-an-economic-firewall"]),
    ("Agent Identity", ["/agent-api", "/agent-capability", "/revocable", "macaroon", "zero-trust"]),
    ("Paid-Rail Governance", ["/l402", "/paid-agent", "http-402", "api-monetization"]),
    ("Blog", ["/blog/"]),
]

INSPECTION_PRIORITY = [
    "/",
    "/tools",
    "/economic-firewall",
    "/ai-agent-cost-control",
    "/mcp-governance",
    "/mcp-budget-enforcement",
    "/mcp-cost-control",
    "/agent-spend-policy-template",
    "/mcp-proxy-config-generator",
    "/agent-api-key-risk-assessment",
    "/l402-api-pricing-calculator",
    "/ai-agent-runaway-spend-benchmark",
    "/compare",
    "/integrations",
]


def iso_date(days_ago: int) -> str:
    return (dt.date.today() - dt.timedelta(days=days_ago)).isoformat()


def get_creds() -> Credentials:
    if not TOKEN_PATH.exists():
        raise SystemExit(f"Missing Search Console token: {TOKEN_PATH}")
    creds = Credentials.from_authorized_user_file(str(TOKEN_PATH), SCOPES)
    if creds.expired and creds.refresh_token:
        creds.refresh(Request())
        TOKEN_PATH.write_text(creds.to_json())
        TOKEN_PATH.chmod(0o600)
    if not creds.valid:
        raise SystemExit("Search Console credentials are invalid. Re-run OAuth setup.")
    return creds


def svc(name: str, version: str, creds: Credentials):
    return build(name, version, credentials=creds, cache_discovery=False)


def fetch_sitemap_urls() -> list[str]:
    with urllib.request.urlopen(SITEMAP_URL, timeout=30) as res:
        raw = res.read()
    root = ET.fromstring(raw)
    ns = {"sm": "http://www.sitemaps.org/schemas/sitemap/0.9"}
    urls = [el.text.strip() for el in root.findall(".//sm:loc", ns) if el.text]
    return sorted(set(urls))


def cluster_for(url_or_path: str) -> str:
    path = url_or_path.replace(BASE_URL, "") or "/"
    for name, needles in CLUSTERS:
        if any(n in path for n in needles):
            return name
    return "Core/Other"


def query_searchanalytics(webmasters, start: str, end: str, dimensions: list[str], row_limit: int = 25000) -> list[dict[str, Any]]:
    body = {
        "startDate": start,
        "endDate": end,
        "dimensions": dimensions,
        "rowLimit": row_limit,
        "startRow": 0,
    }
    rows: list[dict[str, Any]] = []
    while True:
        resp = webmasters.searchanalytics().query(siteUrl=SITE, body=body).execute()
        batch = resp.get("rows", [])
        rows.extend(batch)
        if len(batch) < row_limit:
            return rows
        body["startRow"] += row_limit


def summarize_clusters(page_rows: list[dict[str, Any]]) -> dict[str, dict[str, float]]:
    clusters: dict[str, dict[str, float]] = defaultdict(lambda: {"clicks": 0.0, "impressions": 0.0, "position_weight": 0.0})
    for row in page_rows:
        page = row["keys"][0]
        c = cluster_for(page)
        impressions = float(row.get("impressions", 0) or 0)
        clusters[c]["clicks"] += float(row.get("clicks", 0) or 0)
        clusters[c]["impressions"] += impressions
        clusters[c]["position_weight"] += float(row.get("position", 0) or 0) * impressions
    for data in clusters.values():
        imps = data["impressions"] or 1
        data["ctr"] = data["clicks"] / imps
        data["position"] = data["position_weight"] / imps
    return dict(clusters)


def inspect_urls(searchconsole, urls: list[str]) -> list[dict[str, str]]:
    results = []
    for url in urls:
        try:
            resp = searchconsole.urlInspection().index().inspect(
                body={"inspectionUrl": url, "siteUrl": SITE, "languageCode": "en-US"}
            ).execute()
            status = resp.get("inspectionResult", {}).get("indexStatusResult", {})
            results.append({
                "url": url,
                "verdict": status.get("verdict", "UNKNOWN"),
                "coverageState": status.get("coverageState", "UNKNOWN"),
                "robotsTxtState": status.get("robotsTxtState", "UNKNOWN"),
                "indexingState": status.get("indexingState", "UNKNOWN"),
                "lastCrawlTime": status.get("lastCrawlTime", ""),
            })
        except Exception as exc:  # quota/API errors should not kill the report
            results.append({"url": url, "verdict": "ERROR", "coverageState": str(exc)[:240]})
    return results


def pct(x: float) -> str:
    return f"{x * 100:.2f}%"


def table(headers: list[str], rows: list[list[Any]]) -> str:
    if not rows:
        return "_No rows._\n"
    out = ["| " + " | ".join(headers) + " |", "| " + " | ".join(["---"] * len(headers)) + " |"]
    for row in rows:
        out.append("| " + " | ".join(str(v).replace("|", "\\|") for v in row) + " |")
    return "\n".join(out) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--start", default=iso_date(31), help="Start date YYYY-MM-DD. Default: 31 days ago.")
    parser.add_argument("--end", default=iso_date(3), help="End date YYYY-MM-DD. Default: 3 days ago for GSC freshness lag.")
    parser.add_argument("--out-dir", default=str(DEFAULT_OUT_DIR))
    parser.add_argument("--json-dir", default=str(DEFAULT_JSON_DIR))
    parser.add_argument("--inspect", type=int, default=20, help="Max URLs to inspect. Keep conservative for quota.")
    args = parser.parse_args()

    creds = get_creds()
    webmasters = svc("webmasters", "v3", creds)
    searchconsole = svc("searchconsole", "v1", creds)

    sites = webmasters.sites().list().execute().get("siteEntry", [])
    target = [s for s in sites if s.get("siteUrl") == SITE]
    if not target:
        raise SystemExit(f"No access to {SITE}")

    sitemap_urls = fetch_sitemap_urls()
    page_rows = query_searchanalytics(webmasters, args.start, args.end, ["page"])
    query_rows = query_searchanalytics(webmasters, args.start, args.end, ["query"])
    page_query_rows = query_searchanalytics(webmasters, args.start, args.end, ["page", "query"])

    cluster_summary = summarize_clusters(page_rows)
    page_by_url = {r["keys"][0]: r for r in page_rows}

    # Top-line opportunities.
    low_ctr = [r for r in page_rows if r.get("impressions", 0) >= 25 and r.get("ctr", 0) < 0.02]
    low_ctr.sort(key=lambda r: (-r.get("impressions", 0), r.get("position", 999)))

    striking_distance = [r for r in page_query_rows if r.get("impressions", 0) >= 10 and 4 <= r.get("position", 999) <= 20]
    striking_distance.sort(key=lambda r: (r.get("position", 999), -r.get("impressions", 0)))

    new_priority = [BASE_URL + p for p in INSPECTION_PRIORITY]
    # Add top sitemap URLs not receiving impressions, prioritizing new tools/category pages.
    no_impression = [u for u in sitemap_urls if u not in page_by_url]
    preferred = [u for u in no_impression if any(n in u for n in ["generator", "template", "calculator", "assessment", "mcp", "agent-"])]
    inspect_targets = []
    for u in new_priority + preferred + sitemap_urls:
        if u not in inspect_targets:
            inspect_targets.append(u)
        if len(inspect_targets) >= args.inspect:
            break
    inspection = inspect_urls(searchconsole, inspect_targets)

    total_clicks = sum(r.get("clicks", 0) for r in page_rows)
    total_impressions = sum(r.get("impressions", 0) for r in page_rows)
    avg_position = (sum(r.get("position", 0) * r.get("impressions", 0) for r in page_rows) / total_impressions) if total_impressions else 0
    overall_ctr = total_clicks / total_impressions if total_impressions else 0

    today = dt.datetime.now().strftime("%Y-%m-%d %H:%M")
    report = [
        "---",
        f"title: SatGate SEO Telemetry — {args.start} to {args.end}",
        f"created: {dt.date.today().isoformat()}",
        "type: gsc-telemetry",
        "project: SatGate SEO Category Dominance",
        "---",
        "",
        f"# SatGate SEO Telemetry — {args.start} to {args.end}",
        "",
        f"Generated: {today}",
        "",
        "## Executive Snapshot",
        "",
        f"- Sitemap URLs: **{len(sitemap_urls)}**",
        f"- GSC pages with impressions: **{len(page_rows)}**",
        f"- Clicks: **{total_clicks:.0f}**",
        f"- Impressions: **{total_impressions:.0f}**",
        f"- CTR: **{pct(overall_ctr)}**",
        f"- Avg position: **{avg_position:.1f}**",
        "",
        "## Cluster Performance",
        "",
        table(
            ["Cluster", "Clicks", "Impressions", "CTR", "Avg position"],
            [[k, f"{v['clicks']:.0f}", f"{v['impressions']:.0f}", pct(v["ctr"]), f"{v['position']:.1f}"] for k, v in sorted(cluster_summary.items(), key=lambda kv: -kv[1]["impressions"])],
        ),
        "## Top Pages",
        "",
        table(
            ["Page", "Clicks", "Impressions", "CTR", "Position", "Cluster"],
            [[r["keys"][0].replace(BASE_URL, ""), f"{r.get('clicks',0):.0f}", f"{r.get('impressions',0):.0f}", pct(r.get("ctr", 0)), f"{r.get('position',0):.1f}", cluster_for(r["keys"][0])] for r in sorted(page_rows, key=lambda r: -r.get("impressions", 0))[:25]],
        ),
        "## Top Queries",
        "",
        table(
            ["Query", "Clicks", "Impressions", "CTR", "Position"],
            [[r["keys"][0], f"{r.get('clicks',0):.0f}", f"{r.get('impressions',0):.0f}", pct(r.get("ctr", 0)), f"{r.get('position',0):.1f}"] for r in sorted(query_rows, key=lambda r: -r.get("impressions", 0))[:30]],
        ),
        "## CTR Rewrite Candidates",
        "",
        "Pages with impressions but weak CTR. These usually need sharper titles/meta, better exact-query alignment, or stronger internal links.",
        "",
        table(
            ["Page", "Clicks", "Impressions", "CTR", "Position"],
            [[r["keys"][0].replace(BASE_URL, ""), f"{r.get('clicks',0):.0f}", f"{r.get('impressions',0):.0f}", pct(r.get("ctr", 0)), f"{r.get('position',0):.1f}"] for r in low_ctr[:20]],
        ),
        "## Striking-Distance Query Opportunities",
        "",
        "Queries/pages ranking roughly positions 4-20. These are the best candidates for targeted content expansion and internal links.",
        "",
        table(
            ["Page", "Query", "Clicks", "Impressions", "CTR", "Position"],
            [[r["keys"][0].replace(BASE_URL, ""), r["keys"][1], f"{r.get('clicks',0):.0f}", f"{r.get('impressions',0):.0f}", pct(r.get("ctr", 0)), f"{r.get('position',0):.1f}"] for r in striking_distance[:30]],
        ),
        "## Index Inspection Sample",
        "",
        table(
            ["URL", "Verdict", "Coverage", "Last crawl"],
            [[r["url"].replace(BASE_URL, ""), r.get("verdict", ""), r.get("coverageState", ""), r.get("lastCrawlTime", "")] for r in inspection],
        ),
        "## Recommended Next Actions",
        "",
        "1. Rewrite titles/meta for the top CTR candidates after checking their dominant queries.",
        "2. Add internal links from indexed/high-impression pages to new tools/templates still unknown or discovered-not-indexed.",
        "3. Expand pages in striking distance with query-matched FAQ sections and examples.",
        "4. Re-run this report weekly; avoid repeated sitemap submissions unless sitemap contents materially change.",
    ]

    out_dir = pathlib.Path(args.out_dir)
    json_dir = pathlib.Path(args.json_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    json_dir.mkdir(parents=True, exist_ok=True)
    slug = f"satgate-seo-telemetry-{dt.date.today().isoformat()}.md"
    report_path = out_dir / slug
    report_path.write_text("\n".join(report) + "\n")

    latest_path = out_dir / "satgate-seo-telemetry-latest.md"
    latest_path.write_text("\n".join(report) + "\n")

    artifact = {
        "site": SITE,
        "start": args.start,
        "end": args.end,
        "generated": today,
        "sitemapUrlCount": len(sitemap_urls),
        "summary": {"clicks": total_clicks, "impressions": total_impressions, "ctr": overall_ctr, "position": avg_position},
        "clusters": cluster_summary,
        "pageRows": page_rows,
        "queryRows": query_rows,
        "pageQueryRows": page_query_rows,
        "inspection": inspection,
    }
    json_path = json_dir / f"satgate-seo-telemetry-{dt.date.today().isoformat()}.json"
    json_path.write_text(json.dumps(artifact, indent=2))

    print(f"report={report_path}")
    print(f"latest={latest_path}")
    print(f"json={json_path}")
    print(f"clicks={total_clicks:.0f} impressions={total_impressions:.0f} ctr={pct(overall_ctr)} position={avg_position:.1f}")


if __name__ == "__main__":
    main()
