#!/usr/bin/env python3
"""SatGate SEO machine: score GSC opportunities, statically audit target pages, and emit recommendations.

Runs without external services so PRs can verify SEO hygiene deterministically. Live GSC exports
can replace seo/snapshots/gsc-current.json; the scoring/reporting layer stays the same.
"""
from __future__ import annotations
import argparse, datetime as dt, json, math, re, sys
from html import escape
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / 'app'
SEO = ROOT / 'seo'
REPORTS = SEO / 'reports'
BASE = 'https://satgate.io'

META_BLOCK_RE = re.compile(r"export\s+const\s+metadata\s*(?::[^=]+)?=\s*\{(.*?)\n\};", re.S)
TITLE_RE = re.compile(r"title:\s*([\"'`])(.+?)\1", re.S)
DESC_RE = re.compile(r"description:\s*([\"'`])(.+?)\1", re.S)
CANON_RE = re.compile(r"alternates:\s*\{\s*canonical:\s*([\"'`])https://satgate\.io([^\"'`]+)\1", re.S)
H1_RE = re.compile(r"<h1[^>]*>(.*?)</h1>", re.S)
H2_RE = re.compile(r"<h2[^>]*>(.*?)</h2>", re.S)
LINK_RE = re.compile(r"href=([\"'])(/[^\"'#?]+)\1")
TEXT_TAG_RE = re.compile(r"<[^>]+>|\{[^}]+\}")
SITEMAP_PATH_RE = re.compile(r"path:\s*'([^']*)'")

RECOMMENDED_META = {
 '/blog/how-to-add-budget-limits-to-openai-api-calls': {
   'title': 'How to Set OpenAI API Budget Limits Per Team',
   'description': 'Set OpenAI API budget limits per team or project before GPT calls run. Stop overspend at the request path and keep Evidence Pack proof.'},
 '/blog/llm-cost-management': {
   'title': 'LLM Cost Management: Control AI Spend Before It Happens',
   'description': 'A practical guide to LLM cost management using authority before execution, budget controls, and Evidence Pack receipts for AI agents.'},
 '/blog/http-402-payment-required-use-cases': {
   'title': 'HTTP 402 Payment Required: API and Agent Use Cases',
   'description': 'HTTP 402 and L402 are paid-rail context. SatGate governs authority before execution and preserves Evidence Packs.'},
 '/blog/macaroon-tokens-vs-api-keys': {
   'title': 'Macaroon Tokens vs API Keys for Agent Access',
   'description': 'Compare macaroon tokens and API keys for scoped authorization, delegated access, and safer AI agent permissions.'},
 '/govern': {
   'title': 'AI Agent Governance: Policy-to-Proof',
   'description': 'Govern AI agents with SatGate: authority before execution, Observe/Control/Prove, MCP governance, paid-rail context, and Evidence Packs.'},
 '/blog/api-gateway-for-ai-agents': {
   'title': 'API Gateway for AI Agents: Control Tool and API Access',
   'description': 'Learn how an API gateway for AI agents can enforce authority before execution, budgets, MCP governance, and Evidence Packs across APIs and paid rails.'},
 '/mcp-gateway': {
   'title': 'MCP Gateway for Agent Governance and Evidence Packs',
   'description': 'Use SatGate as an MCP gateway to check authority before tool execution, enforce policy, and export Evidence Packs.'},
 '/capability-auth': {
   'title': 'Capability-Based Authorization for AI Agents',
   'description': 'Replace broad API keys with scoped, revocable, budget-aware capabilities for AI agents using SatGate.'},
}

def load_json(path: Path) -> Any:
    return json.loads(path.read_text())

def route_file(path: str) -> Path:
    return APP / path.strip('/') / 'page.tsx' if path != '/' else APP / 'page.tsx'

def clean_text(raw: str) -> str:
    raw = re.sub(r"<[^>]+>", " ", raw)
    raw = re.sub(r"\{[^}]+\}", " ", raw)
    return ' '.join(raw.split())

def extract_first(regex, text, group=2):
    m=regex.search(text)
    return ' '.join(m.group(group).split()) if m else ''

def expected_ctr(position: float) -> float:
    if position <= 1: return 0.28
    if position <= 2: return 0.15
    if position <= 3: return 0.10
    if position <= 5: return 0.06
    if position <= 8: return 0.035
    if position <= 10: return 0.025
    if position <= 20: return 0.012
    return 0.005

def score_item(row: dict[str, Any], target: dict[str, Any]) -> dict[str, Any]:
    imps=float(row.get('impressions',0) or 0); ctr=float(row.get('ctr',0) or 0); pos=float(row.get('position',100) or 100)
    exp=expected_ctr(pos); gap=max(exp-ctr,0)
    impression_score=math.log10(imps+1)*20 if imps else 0
    ranking_boost=5 if pos<=3 else 30 if pos<=10 else 15 if pos<=20 else 5
    new_boost=20 if target.get('status')=='new' else 0
    strategic_boost=25 if target.get('path') in ['/mcp-gateway','/capability-auth'] else 0
    score=round(impression_score + gap*100 + ranking_boost + new_boost + strategic_boost)
    priority='P0' if score>=90 else 'P1' if score>=70 else 'P2' if score>=50 else 'P3'
    return {**target, 'clicks': row.get('clicks',0), 'impressions': int(imps), 'ctr': ctr, 'position': pos,
            'expectedCtr': exp, 'ctrGap': round(gap,4), 'score': score, 'priority': priority}

def comparison_title_for_path(path: str) -> str:
    if not path.startswith('/compare/'):
        return ''
    config = APP / 'compare' / '_components' / 'comparisons.ts'
    if not config.exists():
        return ''
    slug = path.rsplit('/', 1)[-1]
    text = config.read_text(errors='ignore')
    m = re.search(rf"['\"]{re.escape(slug)}['\"]:\s*\{{.*?title:\s*['\"]([^'\"]+)['\"]", text, re.S)
    return m.group(1) if m else ''

def audit_page(target: dict[str, Any], sitemap_paths: set[str]) -> dict[str, Any]:
    path=target['path']; f=route_file(path); issues=[]
    if not f.exists():
        return {'path': path, 'exists': False, 'issues': ['missing route file'], 'priority': 'P0' if target.get('status')=='new' else 'P1'}
    text=f.read_text(errors='ignore')
    if 'CapabilityLifecycleDemo' in text:
        component = APP / 'capability-lifecycle-demo' / 'CapabilityLifecycleDemo.tsx'
        if component.exists():
            text += '\n' + component.read_text(errors='ignore')
    if 'BrutalComparisonPage' in text:
        component = APP / 'compare' / '_components' / 'BrutalComparisonPage.tsx'
        config = APP / 'compare' / '_components' / 'comparisons.ts'
        if component.exists():
            text += '\n' + component.read_text(errors='ignore')
        if config.exists():
            text += '\n' + config.read_text(errors='ignore')
    if path == '/govern':
        client = APP / 'components' / 'GovernClient.tsx'
        if client.exists():
            text += '\n' + client.read_text(errors='ignore')
    block=META_BLOCK_RE.search(text)
    meta=block.group(1) if block else ''
    title=extract_first(TITLE_RE, meta)
    desc=extract_first(DESC_RE, meta)
    canon_match=CANON_RE.search(meta)
    canonical=canon_match.group(2) if canon_match else ''
    h1s=[clean_text(m.group(1)) for m in H1_RE.finditer(text)]
    comparison_title = comparison_title_for_path(path)
    if comparison_title:
        h1s=[comparison_title if not h1 else h1 for h1 in h1s]
    h2s=[clean_text(m.group(1)) for m in H2_RE.finditer(text)]
    links=sorted({m.group(2) for m in LINK_RE.finditer(text)})
    schema_types=sorted(set(re.findall(r"['\"]@type['\"]:\s*['\"]([^'\"]+)['\"]", text)))
    plain=clean_text(text + '\n' + title + '\n' + desc)
    if not title: issues.append('missing title')
    elif not (35 <= len(title) <= 70): issues.append(f'title length {len(title)} outside 35-70')
    if not desc: issues.append('missing meta description')
    elif not (90 <= len(desc) <= 160): issues.append(f'meta description length {len(desc)} outside 90-160')
    if canonical != path: issues.append(f'canonical mismatch: {canonical or "missing"}')
    if len(h1s) != 1: issues.append(f'h1 count {len(h1s)}')
    if len(h2s) < 3: issues.append(f'only {len(h2s)} h2s')
    if len(links) < 3: issues.append('fewer than 3 internal links')
    if path not in sitemap_paths: issues.append('missing from sitemap')
    if 'FAQPage' not in schema_types: issues.append('missing FAQPage schema')
    if not any(t in schema_types for t in ['TechArticle','Article','WebPage','SoftwareApplication']): issues.append('missing primary structured data')
    if not all(word.lower() in plain.lower() for word in ['Observe','Control','Prove']): issues.append('missing Observe/Control/Prove language')
    if not any(term.lower() in plain.lower() for term in ['Evidence Pack', 'Evidence Packs', 'receipt']): issues.append('missing Evidence Pack/receipt language')
    if not re.search(r'Get Started|Start Free|See SatGate|Generate|Book|Contact|Launch|Use SatGate|Design Partner|Work with SatGate|Learn how|Try', plain, re.I): issues.append('missing product CTA language')
    return {'path': path, 'exists': True, 'title': title, 'titleLength': len(title), 'metaDescription': desc,
            'metaDescriptionLength': len(desc), 'canonical': canonical, 'h1': h1s, 'h1Count': len(h1s),
            'h2s': h2s[:12], 'internalLinks': links, 'schemaTypes': schema_types, 'wordCount': len(plain.split()),
            'hasObserveControlProve': all(word.lower() in plain.lower() for word in ['Observe','Control','Prove']),
            'hasEvidencePack': any(term.lower() in plain.lower() for term in ['Evidence Pack', 'Evidence Packs', 'receipt']),
            'hasFAQ': 'FAQPage' in schema_types, 'hasProductCTA': bool(re.search(r'Get Started|Start Free|See SatGate|Generate|Book|Contact|Launch|Use SatGate|Design Partner|Work with SatGate|Learn how|Try', plain, re.I)),
            'inSitemap': path in sitemap_paths, 'issues': issues}

def recommendation(item: dict[str, Any], audit: dict[str, Any], link_map: dict[str, list[str]]) -> dict[str, Any]:
    path=item['path']; meta=RECOMMENDED_META.get(path,{})
    changes=[]
    if item['priority'] in ['P0','P1']:
        changes.append('Add or tighten the above-the-fold direct-answer block for the primary query.')
        changes.append('Route the first CTA into a tool, signup, or commercial product page instead of letting the article dead-end.')
    if audit.get('issues'):
        changes.append('Fix SEO audit issues: ' + '; '.join(audit['issues']))
    if path == '/mcp-gateway':
        changes.append('Make this the commercial MCP gateway hub and link all MCP blog/tool pages into it.')
    if path == '/capability-auth':
        changes.append('Use identity-vs-capability framing and link macaroon/API-key content into this page.')
    return {'path': path, 'priority': item['priority'], 'score': item['score'], 'primaryKeyword': item['primaryKeyword'],
            'recommendedTitle': meta.get('title',''), 'recommendedMetaDescription': meta.get('description',''),
            'contentChanges': changes, 'internalLinksToAdd': link_map.get(path,[]),
            'schemaToAdd': ['FAQPage','BreadcrumbList'] + (['TechArticle'] if path.startswith('/blog/') else ['SoftwareApplication'])}

def write_md(opps, audits, recs):
    lines=['# SatGate SEO Machine Report', '', f'Generated: {dt.datetime.utcnow().isoformat()}Z', '', '## Ranked opportunities', '']
    for item in opps:
        lines += [f"### {item['priority']} — {item['path']}", f"- Score: {item['score']}", f"- Query: `{item['primaryKeyword']}`", f"- GSC: {item['clicks']} clicks / {item['impressions']} impressions / {item['ctr']*100:.2f}% CTR / pos {item['position']:.2f}", f"- CTR gap: {item['ctrGap']*100:.2f} pp", '']
    lines += ['## Audit failures', '']
    for audit in audits:
        if audit.get('issues'):
            lines += [f"### {audit['path']}"] + [f"- {i}" for i in audit['issues']] + ['']
    lines += ['## Recommendations', '']
    for rec in recs:
        lines += [f"### {rec['path']}", f"- Title: {rec['recommendedTitle']}", f"- Meta: {rec['recommendedMetaDescription']}", '- Content changes:']
        lines += [f"  - {c}" for c in rec['contentChanges']]
        lines += ['- Links to add: ' + ', '.join(rec['internalLinksToAdd']), '']
    return '\n'.join(lines)

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--check', action='store_true'); args=ap.parse_args()
    REPORTS.mkdir(parents=True, exist_ok=True)
    targets=load_json(SEO/'targets.json'); gsc={r['path']: r for r in load_json(SEO/'snapshots/gsc-current.json')}; links=load_json(SEO/'internal-links.json')
    sitemap_text=(APP/'sitemap.ts').read_text(errors='ignore') if (APP/'sitemap.ts').exists() else ''
    sitemap_paths={p or '/' for p in SITEMAP_PATH_RE.findall(sitemap_text)}
    opps=sorted([score_item(gsc.get(t['path'],{}), t) for t in targets], key=lambda x:(x['priority'], -x['score']))
    audits=[audit_page(t, sitemap_paths) for t in targets]
    audit_by={a['path']: a for a in audits}
    recs=[recommendation(o, audit_by.get(o['path'],{}), links) for o in opps]
    (REPORTS/'opportunities.json').write_text(json.dumps({'generatedAt': dt.datetime.utcnow().isoformat()+'Z', 'items': opps}, indent=2)+"\n")
    (REPORTS/'page-audit.json').write_text(json.dumps({'generatedAt': dt.datetime.utcnow().isoformat()+'Z', 'items': audits}, indent=2)+"\n")
    (REPORTS/'recommendations.json').write_text(json.dumps({'generatedAt': dt.datetime.utcnow().isoformat()+'Z', 'items': recs}, indent=2)+"\n")
    md=write_md(opps,audits,recs)
    (REPORTS/'opportunities.md').write_text(md)
    (REPORTS/'recommendations.md').write_text(md)
    failures=[]
    for audit in audits:
        if audit.get('issues'):
            failures.append(f"{audit['path']}: " + '; '.join(audit['issues']))
    print(f"SEO machine wrote {REPORTS.relative_to(ROOT)} with {len(opps)} opportunities and {len(failures)} audit issue groups.")
    if args.check and failures:
        print('\n'.join(failures))
        sys.exit(1)
if __name__ == '__main__': main()
