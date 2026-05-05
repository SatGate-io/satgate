#!/usr/bin/env python3
"""Audit local SatGate SEO routes, links, and snippet metadata.

Checks quiet crawl/answer-engine failure modes:
1. Root-relative links in app/**/*.tsx pointing at missing app/public targets.
2. Placeholder hrefs such as href="#" that hide broken navigation from crawlers.
3. Paths listed in app/sitemap.ts without matching app route or public asset.
4. satgate.io URLs listed in public/llms.txt without matching local targets.
5. Page-level metadata descriptions over 160 characters.
6. Blog article pages missing TechArticle structured data.
7. JSON-LD pages missing dateModified freshness markers.
8. Page canonical URLs that do not match their app route.
9. Page OpenGraph URLs that are missing or do not match their app route.
10. Page metadata exports missing canonical URLs.
11. OpenGraph/Twitter descriptions over 160 characters.
12. Page-level metadata titles over 70 characters.
13. OpenGraph/Twitter titles over 70 characters.
14. Sitemap lastModified dates older than page JSON-LD dateModified dates.
15. Canonical page URLs missing from sitemap.

This stays intentionally lightweight and static so it can run during heartbeat SEO
work without starting Next.js.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import urlparse

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "app"
PUBLIC = ROOT / "public"
SITEMAP = APP / "sitemap.ts"
LLMS = PUBLIC / "llms.txt"
MAX_META_DESCRIPTION_LENGTH = 160
MAX_META_TITLE_LENGTH = 70
MAX_SOCIAL_TITLE_LENGTH = 70

LINK_RE = re.compile(
    r"(?:href|url):\s*['\"](/[^'\"#?]+)['\"]|href=[\"'](/[^\"'#?]+)[\"']"
)
PLACEHOLDER_HREF_RE = re.compile(r"href\s*=\s*['\"]#['\"]")
SITEMAP_ENTRY_RE = re.compile(r"\{\s*path:\s*'([^']*)',\s*lastModified:\s*'([^']*)'", re.DOTALL)
SITEMAP_PATH_RE = re.compile(r"path:\s*'([^']*)'")
LLMS_URL_RE = re.compile(r"https://satgate\.io([^\s)>,]+)")
METADATA_BLOCK_RE = re.compile(r"export\s+const\s+metadata\s*=\s*\{(.*?)\n\};", re.DOTALL)
METADATA_EXPORT_RE = re.compile(r"export\s+const\s+metadata\s*(?::[^=]+)?=", re.DOTALL)
META_TITLE_RE = re.compile(r"title:\s*(['\"])(.*?)\1", re.DOTALL)
META_DESCRIPTION_RE = re.compile(r"description:\s*(['\"])(.*?)\1", re.DOTALL)
TECHARTICLE_RE = re.compile(r"['\"]@type['\"]:\s*['\"]TechArticle['\"]")
JSONLD_RE = re.compile(r"application/ld\+json")
DATE_MODIFIED_RE = re.compile(r"dateModified")
DATE_MODIFIED_VALUE_RE = re.compile(r"dateModified:\s*['\"]([^'\"]+)['\"]")
CANONICAL_RE = re.compile(r"alternates:\s*\{\s*canonical:\s*['\"]https://satgate\.io([^'\"]*)['\"]", re.DOTALL)
OPENGRAPH_BLOCK_RE = re.compile(r"openGraph:\s*\{(.*?)\n\s*\},", re.DOTALL)
OPENGRAPH_URL_RE = re.compile(r"url:\s*['\"]https://satgate\.io([^'\"]*)['\"]")
TWITTER_BLOCK_RE = re.compile(r"twitter:\s*\{(.*?)\n\s*\},", re.DOTALL)
SOCIAL_DESCRIPTION_RE = re.compile(r"description:\s*(['\"])(.*?)\1", re.DOTALL)

IGNORE_PREFIXES = ("/api/",)
IGNORE_PATHS = {"/", ""}
SPECIAL_APP_ROUTES = {
    "/robots.txt": APP / "robots.ts",
    "/sitemap.xml": APP / "sitemap.ts",
}


def normalize_path(path: str) -> str:
    parsed = urlparse(path)
    normalized = parsed.path or "/"
    if normalized != "/" and normalized.endswith("/"):
        normalized = normalized.rstrip("/")
    return normalized


def normalized_text(value: str) -> str:
    return " ".join(value.split())


def target_candidates(path: str) -> list[Path]:
    path = normalize_path(path)
    if path in SPECIAL_APP_ROUTES:
        return [SPECIAL_APP_ROUTES[path]]

    stripped = path.strip("/")
    return [
        APP / stripped / "page.tsx",
        PUBLIC / stripped,
        PUBLIC / stripped / "index.html",
    ]


def route_exists(path: str) -> bool:
    path = normalize_path(path)
    if path in IGNORE_PATHS or path.startswith(IGNORE_PREFIXES):
        return True
    return any(candidate.exists() for candidate in target_candidates(path))


def audit_internal_links() -> tuple[int, list[tuple[str, str]]]:
    missing: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/*.tsx")):
        text = file_path.read_text(errors="ignore")
        links = {match[0] or match[1] for match in LINK_RE.findall(text)}
        scanned += len(links)
        for link in sorted(links):
            if not route_exists(link):
                missing.append((str(file_path.relative_to(ROOT)), link))

    return scanned, missing


def audit_placeholder_hrefs() -> tuple[int, list[tuple[str, str]]]:
    placeholders: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/*.tsx")):
        text = file_path.read_text(errors="ignore")
        matches = list(PLACEHOLDER_HREF_RE.finditer(text))
        scanned += len(matches)
        for match in matches:
            line_number = text.count("\n", 0, match.start()) + 1
            placeholders.append((str(file_path.relative_to(ROOT)), f"line {line_number}: {match.group(0)}"))

    return scanned, placeholders


def sitemap_entries() -> dict[str, str]:
    if not SITEMAP.exists():
        return {}
    return {normalize_path(path): last_modified for path, last_modified in SITEMAP_ENTRY_RE.findall(SITEMAP.read_text(errors="ignore"))}


def audit_sitemap_routes() -> tuple[int, list[tuple[str, str]]]: 
    if not SITEMAP.exists():
        return 0, [(str(SITEMAP.relative_to(ROOT)), "missing sitemap.ts")]

    paths = SITEMAP_PATH_RE.findall(SITEMAP.read_text(errors="ignore"))
    missing: list[tuple[str, str]] = []
    for path in paths:
        if not route_exists(path):
            expected = " or ".join(str(candidate.relative_to(ROOT)) for candidate in target_candidates(path))
            missing.append((path or "/", expected))
    return len(paths), missing


def audit_llms_urls() -> tuple[int, list[tuple[str, str]]]:
    if not LLMS.exists():
        return 0, []

    paths = sorted({normalize_path(path) for path in LLMS_URL_RE.findall(LLMS.read_text(errors="ignore"))})
    missing: list[tuple[str, str]] = []
    for path in paths:
        if not route_exists(path):
            expected = " or ".join(str(candidate.relative_to(ROOT)) for candidate in target_candidates(path))
            missing.append((path, expected))
    return len(paths), missing


def audit_meta_descriptions() -> tuple[int, list[tuple[str, str]]]:
    overlong: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        metadata_block = METADATA_BLOCK_RE.search(text)
        if not metadata_block:
            continue
        match = META_DESCRIPTION_RE.search(metadata_block.group(1))
        if not match:
            continue
        scanned += 1
        description = normalized_text(match.group(2))
        if len(description) > MAX_META_DESCRIPTION_LENGTH:
            overlong.append(
                (
                    str(file_path.relative_to(ROOT)),
                    f"{len(description)} chars: {description}",
                )
            )

    return scanned, overlong


def audit_meta_titles() -> tuple[int, list[tuple[str, str]]]:
    overlong: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        metadata_block = METADATA_BLOCK_RE.search(text)
        if not metadata_block:
            continue
        match = META_TITLE_RE.search(metadata_block.group(1))
        if not match:
            continue
        scanned += 1
        title = normalized_text(match.group(2))
        if len(title) > MAX_META_TITLE_LENGTH:
            overlong.append((str(file_path.relative_to(ROOT)), f"{len(title)} chars: {title}"))

    return scanned, overlong


def audit_blog_techarticles() -> tuple[int, list[tuple[str, str]]]:
    missing: list[tuple[str, str]] = []
    scanned = 0

    blog_root = APP / "blog"
    for file_path in sorted(blog_root.glob("*/page.tsx")):
        scanned += 1
        text = file_path.read_text(errors="ignore")
        if not TECHARTICLE_RE.search(text):
            missing.append((str(file_path.relative_to(ROOT)), "missing TechArticle JSON-LD"))

    return scanned, missing


def audit_jsonld_freshness() -> tuple[int, list[tuple[str, str]]]:
    missing: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        if not JSONLD_RE.search(text):
            continue
        scanned += 1
        if not DATE_MODIFIED_RE.search(text):
            missing.append((str(file_path.relative_to(ROOT)), "missing dateModified near JSON-LD structured data"))

    return scanned, missing


def app_route_for_page(file_path: Path) -> str:
    relative = file_path.relative_to(APP)
    route_parts = relative.parts[:-1]
    if not route_parts:
        return "/"
    return "/" + "/".join(route_parts)


def audit_canonicals() -> tuple[int, list[tuple[str, str]]]:
    mismatches: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        match = CANONICAL_RE.search(text)
        if not match:
            continue
        scanned += 1
        canonical_path = normalize_path(match.group(1))
        route_path = app_route_for_page(file_path)
        if canonical_path != route_path:
            mismatches.append((str(file_path.relative_to(ROOT)), f"canonical {canonical_path} does not match route {route_path}"))

    return scanned, mismatches


def audit_opengraph_urls() -> tuple[int, list[tuple[str, str]]]:
    problems: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        block_match = OPENGRAPH_BLOCK_RE.search(text)
        if not block_match:
            continue
        scanned += 1
        route_path = app_route_for_page(file_path)
        url_match = OPENGRAPH_URL_RE.search(block_match.group(1))
        if not url_match:
            problems.append((str(file_path.relative_to(ROOT)), "missing openGraph.url"))
            continue
        og_path = normalize_path(url_match.group(1))
        if og_path != route_path:
            problems.append((str(file_path.relative_to(ROOT)), f"openGraph.url {og_path} does not match route {route_path}"))

    return scanned, problems


def audit_canonical_presence() -> tuple[int, list[tuple[str, str]]]:
    missing: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        if not METADATA_EXPORT_RE.search(text):
            continue
        scanned += 1
        if not CANONICAL_RE.search(text):
            missing.append((str(file_path.relative_to(ROOT)), "metadata export missing alternates.canonical"))

    return scanned, missing


def audit_social_descriptions() -> tuple[int, list[tuple[str, str]]]:
    overlong: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        for label, block_re in (("openGraph", OPENGRAPH_BLOCK_RE), ("twitter", TWITTER_BLOCK_RE)):
            block_match = block_re.search(text)
            if not block_match:
                continue
            desc_match = SOCIAL_DESCRIPTION_RE.search(block_match.group(1))
            if not desc_match:
                continue
            scanned += 1
            description = normalized_text(desc_match.group(2))
            if len(description) > MAX_META_DESCRIPTION_LENGTH:
                overlong.append((str(file_path.relative_to(ROOT)), f"{label} {len(description)} chars: {description}"))

    return scanned, overlong


def audit_social_titles() -> tuple[int, list[tuple[str, str]]]:
    overlong: list[tuple[str, str]] = []
    scanned = 0

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        for label, block_re in (("openGraph", OPENGRAPH_BLOCK_RE), ("twitter", TWITTER_BLOCK_RE)):
            block_match = block_re.search(text)
            if not block_match:
                continue
            title_match = META_TITLE_RE.search(block_match.group(1))
            if not title_match:
                continue
            scanned += 1
            title = normalized_text(title_match.group(2))
            if len(title) > MAX_SOCIAL_TITLE_LENGTH:
                overlong.append((str(file_path.relative_to(ROOT)), f"{label} {len(title)} chars: {title}"))

    return scanned, overlong


def audit_canonicals_in_sitemap() -> tuple[int, list[tuple[str, str]]]:
    missing: list[tuple[str, str]] = []
    scanned = 0
    sitemap_paths = set(sitemap_entries())

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        match = CANONICAL_RE.search(text)
        if not match:
            continue
        scanned += 1
        canonical_path = normalize_path(match.group(1))
        if canonical_path not in sitemap_paths:
            missing.append((str(file_path.relative_to(ROOT)), f"canonical {canonical_path} missing from sitemap"))

    return scanned, missing


def audit_sitemap_freshness() -> tuple[int, list[tuple[str, str]]]:
    stale: list[tuple[str, str]] = []
    scanned = 0
    entries = sitemap_entries()

    for file_path in sorted(APP.glob("**/page.tsx")):
        text = file_path.read_text(errors="ignore")
        dates = DATE_MODIFIED_VALUE_RE.findall(text)
        if not dates:
            continue
        route_path = app_route_for_page(file_path)
        sitemap_date = entries.get(route_path)
        if not sitemap_date:
            continue
        scanned += 1
        schema_date = max(dates)
        if sitemap_date < schema_date:
            stale.append((str(file_path.relative_to(ROOT)), f"sitemap {sitemap_date} older than schema {schema_date}"))

    return scanned, stale


def main() -> int:
    link_count, missing_links = audit_internal_links()
    placeholder_href_count, placeholder_hrefs = audit_placeholder_hrefs()
    sitemap_count, missing_sitemap_routes = audit_sitemap_routes()
    llms_count, missing_llms_urls = audit_llms_urls()
    meta_description_count, overlong_meta_descriptions = audit_meta_descriptions()
    meta_title_count, overlong_meta_titles = audit_meta_titles()
    blog_techarticle_count, missing_blog_techarticles = audit_blog_techarticles()
    jsonld_freshness_count, missing_jsonld_freshness = audit_jsonld_freshness()
    canonical_count, canonical_mismatches = audit_canonicals()
    opengraph_count, opengraph_problems = audit_opengraph_urls()
    canonical_presence_count, missing_canonical_presence = audit_canonical_presence()
    social_description_count, overlong_social_descriptions = audit_social_descriptions()
    social_title_count, overlong_social_titles = audit_social_titles()
    sitemap_freshness_count, stale_sitemap_freshness = audit_sitemap_freshness()
    canonical_sitemap_count, missing_canonicals_in_sitemap = audit_canonicals_in_sitemap()

    print(f"internal links scanned: {link_count}")
    print(f"placeholder hrefs scanned: {placeholder_href_count}")
    print(f"sitemap entries scanned: {sitemap_count}")
    print(f"llms.txt satgate.io URLs scanned: {llms_count}")
    print(f"page metadata descriptions scanned: {meta_description_count}")
    print(f"page metadata titles scanned: {meta_title_count}")
    print(f"blog TechArticle pages scanned: {blog_techarticle_count}")
    print(f"JSON-LD freshness pages scanned: {jsonld_freshness_count}")
    print(f"canonical URLs scanned: {canonical_count}")
    print(f"OpenGraph URL blocks scanned: {opengraph_count}")
    print(f"metadata canonical presence pages scanned: {canonical_presence_count}")
    print(f"social metadata descriptions scanned: {social_description_count}")
    print(f"social metadata titles scanned: {social_title_count}")
    print(f"sitemap freshness pages scanned: {sitemap_freshness_count}")
    print(f"canonical sitemap coverage pages scanned: {canonical_sitemap_count}")

    failed = False
    if missing_links:
        failed = True
        print("missing internal targets:")
        for file_path, link in missing_links:
            print(f"{file_path}: {link}")

    if placeholder_hrefs:
        failed = True
        print("placeholder hrefs:")
        for file_path, detail in placeholder_hrefs:
            print(f"{file_path}: {detail}")

    if missing_sitemap_routes:
        failed = True
        print("missing sitemap route targets:")
        for route, expected in missing_sitemap_routes:
            print(f"{route}: expected {expected}")

    if missing_llms_urls:
        failed = True
        print("missing llms.txt URL targets:")
        for route, expected in missing_llms_urls:
            print(f"{route}: expected {expected}")

    if overlong_meta_descriptions:
        failed = True
        print(f"metadata descriptions over {MAX_META_DESCRIPTION_LENGTH} chars:")
        for file_path, detail in overlong_meta_descriptions:
            print(f"{file_path}: {detail}")

    if overlong_meta_titles:
        failed = True
        print(f"metadata titles over {MAX_META_TITLE_LENGTH} chars:")
        for file_path, detail in overlong_meta_titles:
            print(f"{file_path}: {detail}")

    if missing_blog_techarticles:
        failed = True
        print("blog pages missing TechArticle structured data:")
        for file_path, detail in missing_blog_techarticles:
            print(f"{file_path}: {detail}")

    if missing_jsonld_freshness:
        failed = True
        print("JSON-LD pages missing dateModified freshness markers:")
        for file_path, detail in missing_jsonld_freshness:
            print(f"{file_path}: {detail}")

    if canonical_mismatches:
        failed = True
        print("canonical URLs that do not match app routes:")
        for file_path, detail in canonical_mismatches:
            print(f"{file_path}: {detail}")

    if opengraph_problems:
        failed = True
        print("OpenGraph URL problems:")
        for file_path, detail in opengraph_problems:
            print(f"{file_path}: {detail}")

    if missing_canonical_presence:
        failed = True
        print("metadata exports missing canonical URLs:")
        for file_path, detail in missing_canonical_presence:
            print(f"{file_path}: {detail}")

    if overlong_social_descriptions:
        failed = True
        print(f"social metadata descriptions over {MAX_META_DESCRIPTION_LENGTH} chars:")
        for file_path, detail in overlong_social_descriptions:
            print(f"{file_path}: {detail}")

    if overlong_social_titles:
        failed = True
        print(f"social metadata titles over {MAX_SOCIAL_TITLE_LENGTH} chars:")
        for file_path, detail in overlong_social_titles:
            print(f"{file_path}: {detail}")

    if stale_sitemap_freshness:
        failed = True
        print("sitemap lastModified dates older than JSON-LD dateModified dates:")
        for file_path, detail in stale_sitemap_freshness:
            print(f"{file_path}: {detail}")

    if missing_canonicals_in_sitemap:
        failed = True
        print("canonical page URLs missing from sitemap:")
        for file_path, detail in missing_canonicals_in_sitemap:
            print(f"{file_path}: {detail}")

    if failed:
        return 1

    print("no missing static/app internal, placeholder href, sitemap, llms.txt, metadata-title/description, blog TechArticle, JSON-LD freshness, canonical, OpenGraph URL, metadata canonical-presence, social-title/description, sitemap-freshness, or canonical-sitemap targets found")
    return 0


if __name__ == "__main__":
    sys.exit(main())
