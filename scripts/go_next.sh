#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

echo "[CSN] stopping old processes (ok if none)..."
pkill -f "uvicorn services.operator.api:app" 2>/dev/null || true
pkill -f "/bin/predictor$" 2>/dev/null || true
pkill -f "/bin/decider$" 2>/dev/null || true
sleep 0.5

echo "[CSN] starting Operator API on :9103 ..."
( source .venv/bin/activate && \
  uvicorn services.operator.api:app --host 0.0.0.0 --port 9103 \
  > /tmp/csn-operator.log 2>&1 ) &

echo "[CSN] starting Predictor proxy on :7001 ..."
( ./bin/predictor > /tmp/csn-predictor.log 2>&1 ) &

echo "[CSN] starting Decider on :7002 (dynamic capacity) ..."
( unset CSN_EDGES_UP && ./bin/decider > /tmp/csn-decider.log 2>&1 ) &

sleep 1
echo "[CSN] pids:"
ps -ef | grep -E "operator_api|/bin/predictor$|/bin/decider$" | grep -v grep || true

echo "[CSN] quick health:"
curl -s http://127.0.0.1:9103/metrics | grep ^csn_edges_up || echo "operator metrics not yet ready"
ss -ltnp | grep -E ':7001|:7002|:9103' || true

echo "[CSN] logs: tail -f /tmp/csn-operator.log /tmp/csn-predictor.log /tmp/csn-decider.log"
