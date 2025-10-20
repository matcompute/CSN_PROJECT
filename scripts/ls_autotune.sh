#!/usr/bin/env bash
set -euo pipefail

DECIDER="http://127.0.0.1:9102"
MU_MIN=0.0
MU_MAX=10.0
MU_STEP_UP=0.5
MU_STEP_DOWN=0.1
GAMMA_MIN=0.0
GAMMA_MAX=40.0
GAMMA_STEP=1.0
TARGET=0.10
INTERVAL=3

get_metric () {
  curl -s "$DECIDER/metrics" | awk -v k="$1" '$1==k {print $2}' | tail -n1
}

get_mu () {
  curl -s "$DECIDER/lagrange/get" | python3 -c 'import sys,json; print(json.load(sys.stdin)["mu_slo"])'
}

get_gamma () {
  curl -s "$DECIDER/lagrange/get" | python3 -c 'import sys,json; print(json.load(sys.stdin)["gamma_fair_ms"])'
}

set_lagrange () {
  local mu="$1" gamma="$2"
  curl -s -X POST "$DECIDER/lagrange/set?mu_slo=${mu}&gamma_fair_ms=${gamma}" >/dev/null
}

clamp () {
  python3 - "$1" "$2" "$3" <<'PY'
import sys
v, lo, hi = map(float, sys.argv[1:4])
if v < lo: v = lo
if v > hi: v = hi
print(v)
PY
}

echo "[ls-autotune] starting... target violation=$TARGET"

MU="$(get_mu 2>/dev/null || echo 0)"
GAMMA="$(get_gamma 2>/dev/null || echo 10)"

while true; do
  VIOL="$(get_metric csn_viol_rate)"
  [ -z "$VIOL" ] && VIOL=0

  # adjust mu_slo
  awk "BEGIN{exit !($VIOL > $TARGET)}" && \
    MU=$(python3 -c "print(float($MU)+$MU_STEP_UP)") || \
    MU=$(python3 -c "print(float($MU)-$MU_STEP_DOWN)")
  MU=$(clamp "$MU" "$MU_MIN" "$MU_MAX")

  # adjust gamma_fair gently
  awk "BEGIN{exit !($VIOL > ($TARGET*1.5))}" && \
    GAMMA=$(python3 -c "print(float($GAMMA)+$GAMMA_STEP)") || \
    GAMMA=$(python3 -c "print(float($GAMMA)-$GAMMA_STEP)")
  GAMMA=$(clamp "$GAMMA" "$GAMMA_MIN" "$GAMMA_MAX")

  set_lagrange "$MU" "$GAMMA"
  echo "[ls-autotune] viol=$(printf '%.3f' "$VIOL") mu_slo=$(printf '%.2f' "$MU") gamma=$(printf '%.2f' "$GAMMA")"
  sleep "$INTERVAL"
done
