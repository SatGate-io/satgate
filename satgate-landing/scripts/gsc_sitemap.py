#!/usr/bin/env python3
"""Check or submit the SatGate sitemap in Google Search Console."""

from __future__ import annotations

import argparse
import json
import pathlib
import urllib.request
import xml.etree.ElementTree as ET
from typing import Any

from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from googleapiclient.discovery import build

SITE = "sc-domain:satgate.io"
SITEMAP_URL = "https://satgate.io/sitemap.xml"
TOKEN_PATH = pathlib.Path("/Users/mattdean/.gbrain/google-search-console-tokens.json")
SCOPES = [
    "https://www.googleapis.com/auth/webmasters",
    "openid",
    "https://www.googleapis.com/auth/userinfo.email",
]


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


def fetch_sitemap_count() -> int:
    with urllib.request.urlopen(SITEMAP_URL, timeout=30) as res:
        root = ET.fromstring(res.read())
    ns = {"sm": "http://www.sitemaps.org/schemas/sitemap/0.9"}
    return len([el.text for el in root.findall(".//sm:loc", ns) if el.text])


def summarize_status(status: dict[str, Any]) -> dict[str, Any]:
    contents = (status.get("contents") or [{}])[0]
    return {
        "path": status.get("path", SITEMAP_URL),
        "lastSubmitted": status.get("lastSubmitted", ""),
        "lastDownloaded": status.get("lastDownloaded", ""),
        "isPending": status.get("isPending", False),
        "submittedUrls": contents.get("submitted", ""),
        "indexedUrls": contents.get("indexed", ""),
        "warnings": status.get("warnings", "0"),
        "errors": status.get("errors", "0"),
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--submit", action="store_true", help="Submit the sitemap before reading status.")
    args = parser.parse_args()

    creds = get_creds()
    webmasters = build("webmasters", "v3", credentials=creds, cache_discovery=False)
    sites = webmasters.sites().list().execute().get("siteEntry", [])
    if not any(site.get("siteUrl") == SITE for site in sites):
        raise SystemExit(f"No access to {SITE}")

    if args.submit:
        webmasters.sitemaps().submit(siteUrl=SITE, feedpath=SITEMAP_URL).execute()

    status = webmasters.sitemaps().get(siteUrl=SITE, feedpath=SITEMAP_URL).execute()
    result = summarize_status(status)
    result["liveSitemapUrlCount"] = fetch_sitemap_count()
    result["submittedThisRun"] = args.submit
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
