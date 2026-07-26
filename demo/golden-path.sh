#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080}"
ADMIN_BASE="${ADMIN_BASE:-$API_BASE}"
# Admin/delegation APIs require the tenant UUID. Public data-plane routes
# require the tenant slug via X-SatGate-Tenant; raw X-Tenant-ID is intentionally
# not trusted on the public path in production.
DEFAULT_TENANT_ID=""
DEFAULT_TENANT_SLUG=""
TENANT_ID="${TENANT_ID:-$DEFAULT_TENANT_ID}"
TENANT_SLUG="${TENANT_SLUG:-${TENANT:-$DEFAULT_TENANT_SLUG}}"
ROUTE="${ROUTE:-/budgeted/ai}"
SUBJECT="${SUBJECT:-hermes@satgate-demo}"
POLICY="${POLICY:-satgate-public-demo}"
BUDGET_CREDITS="${BUDGET_CREDITS:-20}"
COST_CREDITS="${COST_CREDITS:-10}"
RUN_ID="golden-demo-$(date -u +%Y%m%dT%H%M%SZ)-$$"
TMPDIR="$(mktemp -d)"
TOKEN_CREATED_ID=""
TOKEN="${SATGATE_BEARER_TOKEN:-}"
SATGATE_ADMIN_TOKEN="${SATGATE_ADMIN_TOKEN:-demo-admin-token-2024}"

cleanup() {
  rm -rf "$TMPDIR"
  if [[ -n "${TOKEN_CREATED_ID}" && -n "${SATGATE_ADMIN_TOKEN:-}" ]]; then
    curl -fsS -X DELETE \
      -H "X-Admin-Token: ${SATGATE_ADMIN_TOKEN}" \
      -H "X-Tenant-ID: ${TENANT_ID}" \
      "${ADMIN_BASE}/cloud/delegation/token/${TOKEN_CREATED_ID}" >/dev/null 2>&1 || true
  fi
  return 0
}
trap cleanup EXIT

need() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1" >&2; exit 2; }
}
need curl
need jq
need python3

sha256_fingerprint() {
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s' "$1" | sha256sum | awk '{print $1}'
  else
    printf '%s' "$1" | shasum -a 256 | awk '{print $1}'
  fi
}

is_local_api_base() {
  case "$API_BASE" in
    http://localhost:*|http://127.0.0.1:*|http://0.0.0.0:*) return 0 ;;
    *) return 1 ;;
  esac
}

should_self_seed_local_tenant() {
  is_local_api_base || return 1
  [[ "${SATGATE_GOLDEN_PATH_SELF_SEED:-auto}" != "false" ]] || return 1
  # Existing bearer tokens are assumed to belong to the caller's explicit tenant.
  [[ -z "${SATGATE_BEARER_TOKEN:-}" ]] || return 1
  [[ "${SATGATE_GOLDEN_PATH_SELF_SEED:-auto}" == "true" ]] && return 0
  [[ "$TENANT_ID" == "$DEFAULT_TENANT_ID" && "$TENANT_SLUG" == "$DEFAULT_TENANT_SLUG" ]]
}

ensure_local_demo_tenant() {
  should_self_seed_local_tenant || return 0

  local unique req resp status dev_token verify_resp session_token me_resp
  unique="$(date -u +%Y%m%dT%H%M%SZ)-$$"
  req="$TMPDIR/local-signup-request.json"
  resp="$TMPDIR/local-signup-response.json"
  verify_resp="$TMPDIR/local-verify-response.json"
  me_resp="$TMPDIR/local-me-response.json"
  jq -n \
    --arg name "Golden Path Demo ${unique}" \
    --arg email "golden-path-${unique}@example.invalid" \
    '{name:$name,email:$email}' > "$req"

  status="$(curl -sS -o "$resp" -w '%{http_code}' \
    -X POST \
    -H "Content-Type: application/json" \
    --data-binary "@$req" \
    "${ADMIN_BASE}/cloud/signup")"
  if [[ "$status" != "202" ]]; then
    echo "FAIL: local demo tenant signup returned HTTP ${status}" >&2
    cat "$resp" >&2
    exit 1
  fi

  dev_token="$(jq -r '.dev_token // empty' "$resp")"
  if [[ -z "$dev_token" ]]; then
    echo "FAIL: loopback signup response missing guarded dev_token" >&2
    jq '{verification_required, has_dev_token:(.dev_token != null)}' "$resp" >&2
    exit 1
  fi

  status="$(curl -sS -o "$verify_resp" -w '%{http_code}' -G \
    --data-urlencode "token=${dev_token}" \
    "${ADMIN_BASE}/cloud/auth/verify")"
  if [[ "$status" != "200" ]]; then
    echo "FAIL: local signup verification returned HTTP ${status}" >&2
    cat "$verify_resp" >&2
    exit 1
  fi
  TENANT_ID="$(jq -r '.tenant_id // empty' "$verify_resp")"
  session_token="$(jq -r '.session_token // empty' "$verify_resp")"
  if [[ -z "$TENANT_ID" || -z "$session_token" ]]; then
    echo "FAIL: verification response missing tenant_id or session_token" >&2
    exit 1
  fi

  status="$(curl -sS -o "$me_resp" -w '%{http_code}' \
    -H "Authorization: Bearer ${session_token}" \
    "${ADMIN_BASE}/cloud/auth/me")"
  if [[ "$status" != "200" ]]; then
    echo "FAIL: authenticated tenant profile returned HTTP ${status}" >&2
    cat "$me_resp" >&2
    exit 1
  fi
  TENANT_SLUG="$(jq -r '.tenant_slug // .tenant // empty' "$me_resp")"
  if [[ -z "$TENANT_SLUG" ]]; then
    echo "FAIL: authenticated tenant profile missing tenant_slug" >&2
    exit 1
  fi
}

issue_token() {
  if [[ -n "$TOKEN" ]]; then
    return
  fi
  if [[ -z "${SATGATE_ADMIN_TOKEN:-}" ]]; then
    cat >&2 <<ERR
Set SATGATE_BEARER_TOKEN to run with an existing delegated token, or set
SATGATE_ADMIN_TOKEN to let this script issue and later delete a short-lived demo token.
ERR
    exit 2
  fi

  local expires_at
  expires_at="$(python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(minutes=15)).isoformat().replace('+00:00', 'Z'))
PY
)"
  local req="$TMPDIR/delegate-request.json"
  local resp="$TMPDIR/delegate-response.json"
  jq -n \
    --arg name "${RUN_ID}" \
    --arg route_prefix "/budgeted/*" \
    --arg expires_at "$expires_at" \
    --argjson budget "$BUDGET_CREDITS" \
    '{name:$name, budget_limit_credits:$budget, expires_at:$expires_at, scope:{routes:[$route_prefix], methods:["GET","POST"], policy_modes:["control"]}, cost_center:"golden-demo", project:"satgate-proof"}' > "$req"

  curl -fsS -X POST \
    -H "Content-Type: application/json" \
    -H "X-Admin-Token: ${SATGATE_ADMIN_TOKEN}" \
    -H "X-Tenant-ID: ${TENANT_ID}" \
    --data-binary "@$req" \
    "${ADMIN_BASE}/cloud/delegation/delegate" > "$resp"

  TOKEN="$(jq -r '.macaroon_token // .token_secret // empty' "$resp")"
  TOKEN_CREATED_ID="$(jq -r '.token.id // .token_id // empty' "$resp")"
  if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
    echo "Delegation API did not return a macaroon token" >&2
    jq '{token_id:(.token.id // .token_id // null), fields:keys}' "$resp" >&2
    exit 1
  fi
}


quote_pricing() {
  local label="$1"
  local body="$TMPDIR/quote-${label}-pricing.json"
  local status
  status="$(curl -sS -o "$body" -w '%{http_code}' \
    -G "${API_BASE}/v1/pricing" \
    --data-urlencode "path=${ROUTE}" \
    --data-urlencode "method=GET" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "X-SatGate-Tenant: ${TENANT_SLUG}")"
  if [[ "$status" != "200" ]]; then
    echo "FAIL: /v1/pricing quote (${label}) returned HTTP ${status}" >&2
    cat "$body" >&2
    exit 1
  fi
  jq -e --argjson expected_cost "$COST_CREDITS" '
    .cost_credits == $expected_cost
    and (.budget_remaining_credits | type == "number")
    and (.budget_total_credits | type == "number")
    and (.calls_affordable | type == "number")
  ' "$body" >/dev/null
}

quote_request() {
  local label="$1"
  local body="$TMPDIR/quote-${label}-request.json"
  local status
  status="$(curl -sS -o "$body" -w '%{http_code}' \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "X-SatGate-Tenant: ${TENANT_SLUG}" \
    -H "X-SatGate-Quote: 1" \
    "${API_BASE}${ROUTE}")"
  if [[ "$status" != "200" ]]; then
    echo "FAIL: X-SatGate-Quote request (${label}) returned HTTP ${status}" >&2
    cat "$body" >&2
    exit 1
  fi
  jq -e --argjson expected_cost "$COST_CREDITS" '
    .cost_credits == $expected_cost
    and (.budget_remaining_credits | type == "number")
    and (.budget_total_credits | type == "number")
    and (.calls_affordable | type == "number")
  ' "$body" >/dev/null
}

assert_quote_no_debit() {
  local before="$TMPDIR/quote-${1}-pricing.json"
  local after="$TMPDIR/quote-${2}-pricing.json"
  local before_remaining after_remaining
  before_remaining="$(jq -r '.budget_remaining_credits' "$before")"
  after_remaining="$(jq -r '.budget_remaining_credits' "$after")"
  if [[ "$before_remaining" != "$after_remaining" ]]; then
    echo "FAIL: quote changed remaining budget (${before_remaining} -> ${after_remaining})" >&2
    exit 1
  fi
}

quote_affordability_summary() {
  local label="$1"
  local body="$TMPDIR/quote-${label}-pricing.json"
  jq -r '"cost=\(.cost_credits) remaining=\(.budget_remaining_credits) total=\(.budget_total_credits) calls_affordable=\(.calls_affordable)"' "$body"
}

call_budgeted_ai() {
  local n="$1"
  local headers="$TMPDIR/headers-$n.txt"
  local body="$TMPDIR/body-$n.json"
  local status
  status="$(curl -sS -o "$body" -D "$headers" -w '%{http_code}' \
    -X POST \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "X-SatGate-Tenant: ${TENANT_SLUG}" \
    -H "X-Request-ID: ${RUN_ID}-${n}" \
    -H "Content-Type: application/json" \
    --data '{"model":"demo","input":"satgate golden path"}' \
    "${API_BASE}${ROUTE}")"

  local remaining cost receipt_id receipt_hash evidence_id evidence_url reason decision
  remaining="$(awk 'BEGIN{IGNORECASE=1} /^X-SatGate-Budget-Remaining-Credits:/ {gsub("\r",""); print $2}' "$headers" | tail -1)"
  [[ -n "$remaining" ]] || remaining="$(awk 'BEGIN{IGNORECASE=1} /^X-Budget-Remaining:/ {gsub("\r",""); print $2}' "$headers" | tail -1)"
  cost="$(awk 'BEGIN{IGNORECASE=1} /^X-SatGate-Budget-Cost-Credits:/ {gsub("\r",""); print $2}' "$headers" | tail -1)"
  [[ -n "$cost" ]] || cost="$COST_CREDITS"
  receipt_id="$(awk 'BEGIN{IGNORECASE=1} /^X-SatGate-Receipt-ID:/ {gsub("\r",""); print $2}' "$headers" | tail -1)"
  receipt_hash="$(awk 'BEGIN{IGNORECASE=1} /^X-SatGate-Receipt-Hash:/ {gsub("\r",""); print $2}' "$headers" | tail -1)"
  evidence_id="$(awk 'BEGIN{IGNORECASE=1} /^X-SatGate-Evidence-Pack-ID:/ {gsub("\r",""); print $2}' "$headers" | tail -1)"
  evidence_url="$(python3 - "$headers" <<'PY'
import re, sys
for line in open(sys.argv[1], encoding='utf-8', errors='ignore'):
    if line.lower().startswith('link:'):
        m = re.search(r'<([^>]+)>;\s*rel="satgate-evidence-pack"', line)
        if m:
            print(m.group(1)); break
PY
)"
  reason="$(jq -r '.receipt.decision_reason // .error // empty' "$body" 2>/dev/null || true)"
  decision="allowed"
  [[ "$status" == "402" ]] && decision="denied"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$n" "$status" "$decision" "${reason:-budget_authorized}" "${remaining:-?}" "${cost:-?}" "${receipt_id:-?}" "${receipt_hash:-?}" "${evidence_url:-${API_BASE}/v1/evidence/${evidence_id}}"
}

ensure_local_demo_tenant
issue_token
CAP_HASH="sha256:$(sha256_fingerprint "$TOKEN" | cut -c1-12)…$(sha256_fingerprint "$TOKEN" | tail -c 7)"

quote_pricing minted_before
quote_request minted
quote_pricing minted_after_quote
assert_quote_no_debit minted_before minted_after_quote

results="$TMPDIR/results.tsv"
{
  call_budgeted_ai 1
  call_budgeted_ai 2
  call_budgeted_ai 3
} > "$results"

statuses="$(cut -f2 "$results" | paste -sd, -)"
if [[ "$statuses" != "200,200,402" ]]; then
  echo "FAIL: expected HTTP sequence 200,200,402 but got ${statuses}" >&2
  cat "$results" >&2
  exit 1
fi

FINAL_RECEIPT_HASH="$(tail -1 "$results" | cut -f8)"
EVIDENCE_URL="$(tail -1 "$results" | cut -f9)"
if [[ -z "$EVIDENCE_URL" || "$EVIDENCE_URL" == "?" ]]; then
  echo "FAIL: missing final evidence URL" >&2
  cat "$results" >&2
  exit 1
fi
pack="$TMPDIR/evidence-pack.json"
pack_status="$(curl -sS -o "$pack" -w '%{http_code}' "$EVIDENCE_URL")"
if [[ "$pack_status" != "200" ]]; then
  echo "FAIL: Evidence Pack fetch returned HTTP ${pack_status}: ${EVIDENCE_URL}" >&2
  cat "$pack" >&2
  exit 1
fi
if [[ -z "$FINAL_RECEIPT_HASH" || "$FINAL_RECEIPT_HASH" == "?" ]]; then
  FINAL_RECEIPT_HASH="$(jq -r '.receipt_hash // .receipts[0].receipt_hash // empty' "$pack")"
fi
if [[ -z "$FINAL_RECEIPT_HASH" || "$FINAL_RECEIPT_HASH" == "null" ]]; then
  echo "FAIL: missing final receipt hash in headers and Evidence Pack" >&2
  cat "$results" >&2
  jq '{schema_version, receipt_hash, receipt0_hash:(.receipts[0].receipt_hash // null)}' "$pack" >&2
  exit 1
fi

quote_pricing exhausted_before
quote_request exhausted
quote_pricing exhausted_after_quote
assert_quote_no_debit exhausted_before exhausted_after_quote
if [[ "$(jq -r '.calls_affordable' "$TMPDIR/quote-exhausted_after_quote-pricing.json")" != "0" ]]; then
  echo "FAIL: expected calls_affordable=0 after budget exhaustion" >&2
  cat "$TMPDIR/quote-exhausted_after_quote-pricing.json" >&2
  exit 1
fi

jq -e '.schema_version == "satgate.evidence_pack.v1" and (.receipt_hash | startswith("sha256:"))' "$pack" >/dev/null

proof="$TMPDIR/proof.txt"
{
  echo "SatGate Golden HTTP Budget Demo"
  echo "Mode: live SatGate gateway + controlled ${ROUTE} simulator"
  echo "Flow: issue → quote-before-spend → consume → enforce → quote-after-exhaustion → prove"
  echo
  echo "Subject: ${SUBJECT}"
  echo "Tenant UUID: ${TENANT_ID}"
  echo "Tenant slug: ${TENANT_SLUG}"
  echo "Policy: ${POLICY}"
  echo "Route: ${ROUTE}"
  echo "Capability: cap_hash=${CAP_HASH}"
  echo "Token: [redacted]"
  echo
  echo "Budget"
  echo "Initial: ${BUDGET_CREDITS} credits"
  echo "Per call: ${COST_CREDITS} credits"
  echo
  echo "Pre-spend quote"
  echo "Pricing API: $(quote_affordability_summary minted_before)"
  echo "X-SatGate-Quote: $(jq -r '"cost=\(.cost_credits) remaining=\(.budget_remaining_credits) total=\(.budget_total_credits) calls_affordable=\(.calls_affordable)"' "$TMPDIR/quote-minted-request.json")"
  echo "No-debit assertion: pricing remaining unchanged after X-SatGate-Quote"
  echo
  echo "Calls"
  while IFS=$'\t' read -r n status decision reason remaining cost receipt_id receipt_hash evidence_url; do
    if [[ "$status" == "200" ]]; then label="allowed"; else label="denied"; fi
    printf '%s. HTTP %s %-7s request_id=%s-%s remaining=%s cost=%s receipt=%s\n' "$n" "$status" "$label" "$RUN_ID" "$n" "$remaining" "$cost" "$receipt_id"
    if [[ "$status" == "402" ]]; then
      echo "   reason=${reason}"
    fi
  done < "$results"
  echo
  echo "Proof"
  echo "Receipt hash: ${FINAL_RECEIPT_HASH}"
  echo "Evidence Pack: ${EVIDENCE_URL}"
  echo "Post-exhaustion quote: $(quote_affordability_summary exhausted_after_quote)"
  echo "Post-exhaustion assertion: calls_affordable=0 and quote remaining unchanged"
  echo "Verify local integrity: python3 verify_evidence_pack.py '${EVIDENCE_URL}' --allow-embedded-key"
  echo "Inspect: curl -fsS '${EVIDENCE_URL}' | jq '.schema_version,.receipt_hash,.decision,.budget_state'"
  echo
  echo "Result"
  echo "PASS: SatGate issued bounded authority, exposed price/affordability before spend without debit, allowed budgeted use, enforced exhaustion, showed calls_affordable=0 after exhaustion, and produced buyer-fetchable signed evidence."
  echo
  echo "Caveat"
  echo "This proves HTTP request-path budget enforcement, HTTP quote affordability, no-debit price discovery, and evidence packaging for a controlled demo route. The local HTTPS issuer identity is self-signed and does not serve a trusted JWKS; --allow-embedded-key proves integrity/self-consistency only and does not establish trust in a canonical SatGate issuer. It does not prove charge/L402 quote support, invoice settlement, third-party model billing reconciliation, MCP behavior, durable evidence retention/export, or general production readiness."
} > "$proof"

if grep -Fq "$TOKEN" "$proof"; then
  echo "FAIL: proof output leaked bearer token" >&2
  exit 1
fi

cat "$proof"
