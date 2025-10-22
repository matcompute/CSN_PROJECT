#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

OP=http://127.0.0.1:9103
DEC_METRICS=http://127.0.0.1:9102/metrics
RUN_DIR="experiments/runs/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RUN_DIR"

is_up() { ss -ltn 2>/dev/null | awk '{print $4}' | grep -q ":$1$"; }

echo "[reproduce] ensuring services are up…"
if ! is_up 7001; then ./bin/predictor & echo $! > "$RUN_DIR/predictor.pid"; sleep 0.3; fi
if ! is_up 8000; then uvicorn ml.serve_predictor:app --host 127.0.0.1 --port 8000 & echo $! > "$RUN_DIR/model.pid"; sleep 0.4; fi
if ! is_up 9103; then uvicorn services.operator.api:app --host 0.0.0.0 --port 9103 & echo $! > "$RUN_DIR/operator.pid"; sleep 0.4; fi
if ! is_up 7002; then ./bin/decider & echo $! > "$RUN_DIR/decider.pid"; sleep 0.5; fi

echo "[reproduce] services:"
sudo lsof -i -P -n | egrep ':8000|:7001|:7002|:9102|:9103' | grep LISTEN | tee "$RUN_DIR/ports.txt" || true

# helpers
edges_up() { curl -s "$OP/metrics" | awk '$1=="csn_edges_up"{print $2}'; }
metric_pair() {
  curl -s "$DEC_METRICS" | awk '
    $1=="csn_viol_rate"{v=$2}
    $1=="csn_explore_epsilon"{e=$2}
    END{printf("viol_rate=%s, epsilon=%s\n", v, e)}'
}
phase() {
  local NAME="$1" OUT="$RUN_DIR/${2}"
  echo -e "\n=== PHASE: $NAME ===" | tee -a "$RUN_DIR/log.txt"
  echo "edges_up=$(edges_up)" | tee -a "$RUN_DIR/log.txt"
  seq 120 | xargs -I{} -P 16 ./bin/invoker | tee "$OUT"
  echo "$(metric_pair)" | tee -a "$RUN_DIR/log.txt"
}

# reset to exactly 2 edges (deterministic)
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e2"}' >/dev/null || true
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e3"}' >/dev/null || true
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e4"}' >/dev/null || true
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e5"}' >/dev/null || true
curl -s -X POST "$OP/edge"  -H 'Content-Type: application/json' -d '{"name":"e2"}' >/dev/null
curl -s -X POST "$OP/edge"  -H 'Content-Type: application/json' -d '{"name":"e3"}' >/dev/null

# run phases
phase "BASELINE (2 edges)"        "phase1.out"
curl -s -X POST "$OP/edge" -H 'Content-Type: application/json' -d '{"name":"e4"}' >/dev/null
curl -s -X POST "$OP/edge" -H 'Content-Type: application/json' -d '{"name":"e5"}' >/dev/null
phase "SCALE UP (4 edges)"        "phase2.out"
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e2"}' >/dev/null
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e3"}' >/dev/null
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e4"}' >/dev/null
phase "SCALE DOWN (1 edge)"       "phase3.out"

# summarize action counts per phase
for f in phase1.out phase2.out phase3.out; do
  echo "---- counts for $f ----" | tee -a "$RUN_DIR/counts.txt"
  awk -F': ' '/Chosen action/ {print $2}' "$RUN_DIR/$f" | awk -F'[ :]' '{print $1}' | sort | uniq -c | tee -a "$RUN_DIR/counts.txt"
done

# snapshot key metrics
curl -s "$DEC_METRICS" | egrep 'csn_viol_rate|csn_explore_epsilon|csn_mu_slo|csn_gamma_fair_ms|csn_drift_' | tee "$RUN_DIR/metrics_snapshot.txt" || true
curl -s "$OP/metrics" | grep '^csn_edges_up' | tee "$RUN_DIR/op_snapshot.txt" || true

echo -e "\n[reproduce] done. Run artifacts in: $RUN_DIR"
echo "[reproduce] quick view:"
cat "$RUN_DIR/log.txt"
echo "[reproduce] action counts:"
cat "$RUN_DIR/counts.txt"
