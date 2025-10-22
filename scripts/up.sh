#!/usr/bin/env bash
set -euo pipefail

need() { ! lsof -i -P -n | egrep -q "$1.*LISTEN"; }

# model server :8000
if need ":8000"; then
  echo "[up] model :8000"
  uvicorn ml.serve_predictor:app --host 127.0.0.1 --port 8000 &>/tmp/csn_model.log &
fi

# predictor proxy :7001
if need ":7001"; then
  echo "[up] predictor :7001"
  ./bin/predictor &>/tmp/csn_predictor.log &
fi

# operator :9103
if need ":9103"; then
  echo "[up] operator :9103"
  uvicorn services.operator.api:app --host 0.0.0.0 --port 9103 &>/tmp/csn_operator.log &
fi

# decider :7002 / :9102
if need ":7002"; then
  echo "[up] decider :7002/:9102"
  ./bin/decider &>/tmp/csn_decider.log &
fi

sleep 1
echo "[up] listening ports:"
sudo lsof -i -P -n | egrep ':8000|:7001|:7002|:9102|:9103' | grep LISTEN || true
