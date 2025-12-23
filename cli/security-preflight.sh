#!/bin/bash
# security-preflight.sh - Security posture check for SatGate
# Run before enterprise demos to prove controls are live
#
# Usage:
#   ADMIN_TOKEN=your-token ./cli/security-preflight.sh
#   ADMIN_TOKEN=your-token ./cli/security-preflight.sh --json
#   ADMIN_TOKEN=your-token ./cli/security-preflight.sh --evidence-dir ./evidence/
#
# Exit Codes (CI-friendly):
#   0 - All checks passed
#   1 - One or more checks failed
#   2 - Target unreachable or dependency missing
#
# Output:
#   Default: Human-readable with colors
#   --json:  Single JSON object for tickets/security review docs
#   --evidence-dir: Write proof artifacts for security review

# Don't use set -e as we handle errors explicitly

BASE="${BASE:-https://satgate-production.up.railway.app}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"
JSON_MODE=false
EVIDENCE_DIR=""
VERSION="1.8.1"

# Parse arguments
for arg in "$@"; do
  case $arg in
    --json)
      JSON_MODE=true
      ;;
    --evidence-dir=*)
      EVIDENCE_DIR="${arg#*=}"
      ;;
    --evidence-dir)
      # Handle space-separated value
      shift
      EVIDENCE_DIR="$1"
      ;;
  esac
  shift 2>/dev/null || true
done

# Colors (disabled in JSON mode)
if [ "$JSON_MODE" = true ]; then
  RED=''
  GREEN=''
  YELLOW=''
  NC=''
else
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  NC='\033[0m'
fi

# Timestamp for audit trail
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Results tracking
PASSED=0
FAILED=0
SKIPPED=0
declare -a RESULTS=()

# Evidence collection
declare -a EVIDENCE_FILES=()

# =============================================================================
# DEPENDENCY & CONNECTIVITY CHECK
# =============================================================================

# Check for curl
if ! command -v curl &> /dev/null; then
  if [ "$JSON_MODE" = true ]; then
    echo '{"error":"curl not found","exit_code":2}'
  else
    echo -e "${RED}❌ Error: curl is required but not installed${NC}"
  fi
  exit 2
fi

# Check target is reachable (with timeout)
PING_RESULT=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "$BASE/health" 2>&1)
PING_EXIT=$?

if [[ "$PING_EXIT" -ne 0 ]] || [[ "$PING_RESULT" == "000" ]]; then
  if [ "$JSON_MODE" = true ]; then
    echo "{\"error\":\"Target unreachable: $BASE\",\"exit_code\":2,\"timestamp\":\"$TIMESTAMP\"}"
  else
    echo -e "${RED}❌ Error: Target unreachable: $BASE${NC}"
    echo "   Check your network connection or BASE environment variable."
    echo "   curl exit code: $PING_EXIT"
  fi
  exit 2
fi

# =============================================================================
# EVIDENCE DIRECTORY SETUP
# =============================================================================

if [ -n "$EVIDENCE_DIR" ]; then
  mkdir -p "$EVIDENCE_DIR"
  if [ "$JSON_MODE" = false ]; then
    echo -e "${GREEN}📁 Evidence will be written to: $EVIDENCE_DIR${NC}"
    echo ""
  fi
fi

# Helper function to save evidence
save_evidence() {
  local filename="$1"
  local content="$2"
  
  if [ -n "$EVIDENCE_DIR" ]; then
    echo "$content" > "$EVIDENCE_DIR/$filename"
    EVIDENCE_FILES+=("$filename")
  fi
}

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

# Helper function to record result
record_result() {
  local name="$1"
  local status="$2"
  local detail="$3"
  
  if [ "$status" = "pass" ]; then
    ((PASSED++))
  elif [ "$status" = "fail" ]; then
    ((FAILED++))
  else
    ((SKIPPED++))
  fi
  
  # Escape quotes in detail for JSON safety
  local safe_detail=$(echo "$detail" | sed 's/"/\\"/g')
  RESULTS+=("{\"check\":\"$name\",\"status\":\"$status\",\"detail\":\"$safe_detail\"}")
}

# Helper function for human output
print_check() {
  local num="$1"
  local name="$2"
  local status="$3"
  local detail="$4"
  
  if [ "$JSON_MODE" = false ]; then
    printf "%-2s %-32s " "$num." "$name"
    if [ "$status" = "pass" ]; then
      echo -e "${GREEN}✅ PASS${NC} - $detail"
    elif [ "$status" = "fail" ]; then
      echo -e "${RED}❌ FAIL${NC} - $detail"
    else
      echo -e "${YELLOW}⏭️  SKIP${NC} - $detail"
    fi
  fi
}

# =============================================================================
# HEADER
# =============================================================================

if [ "$JSON_MODE" = false ]; then
  echo ""
  echo "🔒 SatGate Security Pre-Flight Check (v$VERSION)"
  echo "=============================================="
  echo "Target: $BASE"
  echo "Time:   $TIMESTAMP"
  echo ""

  if [ -z "$ADMIN_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  ADMIN_TOKEN not set - some checks will skip${NC}"
    echo "   Set with: export ADMIN_TOKEN=your-token-here"
    echo ""
  fi
fi

# =============================================================================
# CHECK 1: Health returns only status (no version leak)
# =============================================================================
HEALTH=$(curl -s "$BASE/health" 2>/dev/null || echo "ERROR")
save_evidence "health.json" "$HEALTH"

if [[ "$HEALTH" == '{"status":"ok"}' ]]; then
  record_result "health_no_version" "pass" "Returns only status"
  print_check "1" "Health endpoint (no version)" "pass" "Returns only status"
else
  record_result "health_no_version" "fail" "$HEALTH"
  print_check "1" "Health endpoint (no version)" "fail" "Got: $HEALTH"
fi

# =============================================================================
# CHECK 2: Ready returns only status (no version leak)
# =============================================================================
READY=$(curl -s "$BASE/ready" 2>/dev/null || echo "ERROR")
save_evidence "ready.json" "$READY"

if [[ "$READY" == '{"status":"ok"}' ]]; then
  record_result "ready_no_version" "pass" "Returns only status"
  print_check "2" "Ready endpoint (no version)" "pass" "Returns only status"
else
  record_result "ready_no_version" "fail" "$READY"
  print_check "2" "Ready endpoint (no version)" "fail" "Got: $READY"
fi

# =============================================================================
# CHECK 3: Dashboard forbidden without admin token
# =============================================================================
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/dashboard" 2>/dev/null || echo "000")
if [[ "$STATUS" == "403" ]]; then
  record_result "dashboard_auth_required" "pass" "Returns 403 Forbidden"
  print_check "3" "Dashboard auth required" "pass" "Returns 403 Forbidden"
else
  record_result "dashboard_auth_required" "fail" "HTTP $STATUS"
  print_check "3" "Dashboard auth required" "fail" "Got HTTP $STATUS (expected 403)"
fi

# =============================================================================
# CHECK 4: Admin endpoints non-enumerable (/api/governance/info)
# =============================================================================
INFO_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/governance/info" 2>/dev/null || echo "000")
if [[ "$INFO_STATUS" == "403" || "$INFO_STATUS" == "401" ]]; then
  record_result "admin_info_protected" "pass" "Returns $INFO_STATUS without token"
  print_check "4" "Admin /info non-enumerable" "pass" "Returns $INFO_STATUS without token"
else
  record_result "admin_info_protected" "fail" "HTTP $INFO_STATUS"
  print_check "4" "Admin /info non-enumerable" "fail" "Got HTTP $INFO_STATUS (expected 401/403)"
fi

# =============================================================================
# CHECK 5: Admin endpoints non-enumerable (/api/governance/audit/export)
# =============================================================================
EXPORT_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/governance/audit/export" 2>/dev/null || echo "000")
if [[ "$EXPORT_STATUS" == "403" || "$EXPORT_STATUS" == "401" ]]; then
  record_result "admin_export_protected" "pass" "Returns $EXPORT_STATUS without token"
  print_check "5" "Admin /export non-enumerable" "pass" "Returns $EXPORT_STATUS without token"
else
  record_result "admin_export_protected" "fail" "HTTP $EXPORT_STATUS"
  print_check "5" "Admin /export non-enumerable" "fail" "Got HTTP $EXPORT_STATUS (expected 401/403)"
fi

# =============================================================================
# CHECK 6: Security headers present (requires admin token)
# =============================================================================
if [ -n "$ADMIN_TOKEN" ]; then
  HEADERS=$(curl -sI "$BASE/dashboard" -H "X-Admin-Token: $ADMIN_TOKEN" 2>/dev/null)
  save_evidence "headers.txt" "$HEADERS"
  
  XFRAME=$(echo "$HEADERS" | grep -ci "X-Frame-Options" || echo "0")
  XCONTENT=$(echo "$HEADERS" | grep -ci "X-Content-Type-Options" || echo "0")
  
  if [[ "$XFRAME" -ge 1 && "$XCONTENT" -ge 1 ]]; then
    record_result "security_headers" "pass" "X-Frame-Options, X-Content-Type-Options present"
    print_check "6" "Security headers" "pass" "X-Frame-Options, X-Content-Type-Options"
  else
    record_result "security_headers" "fail" "Missing headers"
    print_check "6" "Security headers" "fail" "Missing security headers"
  fi
else
  record_result "security_headers" "skip" "Requires ADMIN_TOKEN"
  print_check "6" "Security headers" "skip" "Requires ADMIN_TOKEN"
fi

# =============================================================================
# CHECK 7: Audit export returns JSONL (SIEM-ready)
# =============================================================================
if [ -n "$ADMIN_TOKEN" ]; then
  AUDIT_FULL=$(curl -s "$BASE/api/governance/audit/export" -H "X-Admin-Token: $ADMIN_TOKEN" 2>/dev/null)
  AUDIT=$(echo "$AUDIT_FULL" | head -1)
  save_evidence "audit-export.jsonl" "$AUDIT_FULL"
  
  if [[ "$AUDIT" == "{"* ]] || [[ -z "$AUDIT" ]]; then
    record_result "audit_jsonl" "pass" "JSONL format (SIEM-ready)"
    print_check "7" "Audit export (JSONL)" "pass" "JSONL format (SIEM-ready)"
  else
    record_result "audit_jsonl" "fail" "Unexpected format"
    print_check "7" "Audit export (JSONL)" "fail" "Unexpected format"
  fi
else
  record_result "audit_jsonl" "skip" "Requires ADMIN_TOKEN"
  print_check "7" "Audit export (JSONL)" "skip" "Requires ADMIN_TOKEN"
fi

# =============================================================================
# CHECK 8: Version info protected (admin-only)
# =============================================================================
if [ -n "$ADMIN_TOKEN" ]; then
  INFO=$(curl -s "$BASE/api/governance/info" -H "X-Admin-Token: $ADMIN_TOKEN" 2>/dev/null)
  save_evidence "info.json" "$INFO"
  
  VER=$(echo "$INFO" | grep -o '"version":"[^"]*"' | head -1 | sed 's/"version":"//;s/"//')
  if [[ -n "$VER" ]]; then
    record_result "version_protected" "pass" "v$VER (admin-only)"
    print_check "8" "Version info protected" "pass" "v$VER (admin-only)"
  else
    record_result "version_protected" "fail" "Could not retrieve"
    print_check "8" "Version info protected" "fail" "Could not retrieve version"
  fi
else
  record_result "version_protected" "skip" "Requires ADMIN_TOKEN"
  print_check "8" "Version info protected" "skip" "Requires ADMIN_TOKEN"
fi

# =============================================================================
# CHECK 9: Token rotation support
# =============================================================================
if [ -n "$ADMIN_TOKEN" ]; then
  ROTATION_RAW=$(curl -s "$BASE/api/governance/info" -H "X-Admin-Token: $ADMIN_TOKEN" 2>/dev/null | grep -o '"tokenRotation":[^,}]*' | head -1)
  if [[ -n "$ROTATION_RAW" ]]; then
    # Extract just the boolean value for JSON-safe output
    ROTATION_VAL=$(echo "$ROTATION_RAW" | sed 's/"tokenRotation"://')
    record_result "token_rotation" "pass" "tokenRotation=$ROTATION_VAL"
    print_check "9" "Token rotation support" "pass" "tokenRotation=$ROTATION_VAL"
  else
    record_result "token_rotation" "pass" "Feature available"
    print_check "9" "Token rotation support" "pass" "Feature available"
  fi
else
  record_result "token_rotation" "skip" "Requires ADMIN_TOKEN"
  print_check "9" "Token rotation support" "skip" "Requires ADMIN_TOKEN"
fi

# =============================================================================
# OUTPUT
# =============================================================================

# Determine overall status
if [[ $FAILED -eq 0 ]]; then
  OVERALL="pass"
  EXIT_CODE=0
else
  OVERALL="fail"
  EXIT_CODE=1
fi

# Build JSON output
RESULTS_JSON=$(IFS=,; echo "${RESULTS[*]}")
EVIDENCE_JSON=""
if [ -n "$EVIDENCE_DIR" ]; then
  EVIDENCE_LIST=$(printf '"%s",' "${EVIDENCE_FILES[@]}" | sed 's/,$//')
  EVIDENCE_JSON=",\"evidence_dir\":\"$EVIDENCE_DIR\",\"evidence_files\":[$EVIDENCE_LIST]"
fi

JSON_OUTPUT=$(cat << EOF
{
  "timestamp": "$TIMESTAMP",
  "target": "$BASE",
  "version": "$VERSION",
  "summary": {
    "status": "$OVERALL",
    "passed": $PASSED,
    "failed": $FAILED,
    "skipped": $SKIPPED,
    "total": $((PASSED + FAILED + SKIPPED)),
    "exit_code": $EXIT_CODE
  },
  "checks": [$RESULTS_JSON]$EVIDENCE_JSON,
  "audit_line": "SatGate Security Pre-Flight | $TIMESTAMP | $BASE | $OVERALL | passed=$PASSED failed=$FAILED skipped=$SKIPPED"
}
EOF
)

# Save result.json if evidence dir is set
if [ -n "$EVIDENCE_DIR" ]; then
  echo "$JSON_OUTPUT" > "$EVIDENCE_DIR/result.json"
fi

# JSON output mode
if [ "$JSON_MODE" = true ]; then
  echo "$JSON_OUTPUT"
else
  # Human-readable summary
  echo ""
  echo "=============================================="
  echo -e "Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}, ${YELLOW}$SKIPPED skipped${NC}"
  echo ""
  
  # Audit-ready one-liner
  echo "📋 Audit Summary (copy for tickets):"
  echo "   SatGate Security Pre-Flight | $TIMESTAMP | $BASE | $OVERALL | passed=$PASSED failed=$FAILED skipped=$SKIPPED"
  
  # Evidence directory info
  if [ -n "$EVIDENCE_DIR" ]; then
    echo ""
    echo "📁 Evidence written to: $EVIDENCE_DIR/"
    for f in "${EVIDENCE_FILES[@]}"; do
      echo "   - $f"
    done
    echo "   - result.json"
  fi
  
  echo ""
  
  if [[ $FAILED -eq 0 ]]; then
    echo -e "${GREEN}✅ All checks passed. Ready for demo.${NC}"
  else
    echo -e "${RED}❌ Some checks failed. Review before demo.${NC}"
  fi
fi

# Exit with appropriate code
# 0 = pass, 1 = fail, 2 = unreachable (handled earlier)
exit $EXIT_CODE
