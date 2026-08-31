#!/usr/bin/env bash
#
# loadtest_rate_limit.sh
#
# Hammers POST /api/orders on a *running* Velocity instance to verify
# rate limiting is actually enforced end-to-end (config + Redis + middleware
# all wired together correctly) — not just that the unit tests pass.
#
# Usage:
#   VELOCITY_BASE_URL="http://localhost:8080" \
#   VELOCITY_TOKEN="<jwt>" \
#   ./loadtest_rate_limit.sh [request_count]
#
# What to look for:
#   - The first N requests (N = submit_burst in the active config) should
#     return 201/200 (or your handler's validation error, e.g. 400 for a
#     malformed body — anything that ISN'T 429).
#   - Requests after that should start returning 429 with a Retry-After
#     header.
#   - If EVERY request returns non-429 no matter how many you send, rate
#     limiting is not enforced on this environment (this is exactly the
#     staging/prod gap found in review: missing `rate_limit:` config
#     section -> Enabled defaults to false).

set -euo pipefail

BASE_URL="${VELOCITY_BASE_URL:-http://localhost:8080}"
TOKEN="${VELOCITY_TOKEN:-}"
COUNT="${1:-30}"

if [[ -z "$TOKEN" ]]; then
  echo "ERROR: set VELOCITY_TOKEN to a valid JWT for an authenticated user." >&2
  exit 1
fi

# Minimal order payload — adjust fields to match SubmitOrderRequest if
# your schema differs (check internal/transport/http/dto/request/submit_order_request.go).
BODY='{"symbol":"BTCUSDT","side":"buy","type":"limit","price":"50000","quantity":"0.001"}'

echo "Target:   POST ${BASE_URL}/api/orders"
echo "Requests: ${COUNT}"
echo "---------------------------------------------------------------"
printf "%-5s %-6s %-18s %-12s\n" "#" "HTTP" "X-RateLimit-Rem" "Retry-After"

allowed=0
limited=0

for i in $(seq 1 "$COUNT"); do
  resp_headers=$(mktemp)

  status=$(curl -s -o /dev/null -D "$resp_headers" -w "%{http_code}" \
    -X POST "${BASE_URL}/api/orders" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$BODY")

  remaining=$(grep -i '^X-RateLimit-Remaining:' "$resp_headers" | tr -d '\r' | awk '{print $2}')
  retry_after=$(grep -i '^Retry-After:' "$resp_headers" | tr -d '\r' | awk '{print $2}')
  rm -f "$resp_headers"

  printf "%-5s %-6s %-18s %-12s\n" "$i" "$status" "${remaining:--}" "${retry_after:--}"

  if [[ "$status" == "429" ]]; then
    limited=$((limited + 1))
  else
    allowed=$((allowed + 1))
  fi

  # Fire quickly — we want to exceed burst, not just the sustained rate.
  sleep 0.05
done

echo "---------------------------------------------------------------"
echo "Allowed: $allowed   Rate-limited (429): $limited"

if [[ "$limited" -eq 0 ]]; then
  echo
  echo "WARNING: no request was rate-limited across ${COUNT} rapid requests." >&2
  echo "         Either raise \$COUNT further, or rate limiting is disabled" >&2
  echo "         on this environment (check the rate_limit: section of the" >&2
  echo "         active config, and that RateLimitConfig.Enabled == true)." >&2
  exit 2
fi