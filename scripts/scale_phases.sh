#!/usr/bin/env bash
set -euo pipefail

OP="http://127.0.0.1:9103"
DEC_METRICS="http://127.0.0.1:9102/metrics"

phase() {
  echo -e "\n=== PHASE: $1 ==="
  echo "edges_up=$(curl -s $OP/metrics | awk '$1==\"csn_edges_up\"{print $2}')"
  for i in {1..10}; do ./bin/invoker; done
  curl -s "$DEC_METRICS" | egrep 'csn_viol_rate|csn_explore_epsilon' || true
}

# ensure operator up
curl -sf "$OP/metrics" >/dev/null || { echo "Operator not up on 9103"; exit 1; }

# reset to 2 edges (drain all, then add two deterministic)
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e2"}' >/dev/null || true
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e3"}' >/dev/null || true
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"edge4"}' >/dev/null || true
curl -s -X POST "$OP/edge" -H 'Content-Type: application/json' -d '{"name":"e2"}' >/dev/null
curl -s -X POST "$OP/edge" -H 'Content-Type: application/json' -d '{"name":"e3"}' >/dev/null

phase "BASELINE (2 edges)"

# scale up to 4
curl -s -X POST "$OP/edge" -H 'Content-Type: application/json' -d '{"name":"e4"}' >/dev/null
curl -s -X POST "$OP/edge" -H 'Content-Type: application/json' -d '{"name":"e5"}' >/dev/null
phase "SCALE UP (4 edges)"

# scale down to 1
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e2"}' >/dev/null
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e3"}' >/dev/null
curl -s -X POST "$OP/drain" -H 'Content-Type: application/json' -d '{"name":"e4"}' >/dev/null
phase "SCALE DOWN (1 edge)"

echo -e "\n=== Done. Recent telemetry rows ==="
tail -n 6 experiments/telemetry.csv 2>/dev/null || echo "no telemetry file yet"
