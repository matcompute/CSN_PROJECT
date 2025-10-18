SHELL := /bin/bash

PY := .venv/bin/python
PIP := .venv/bin/pip

.PHONY: help venv deps build fmt lint run-fastapi run-predictor run-decider invoker sweep sweep-policies analyze dynamic clean

help:
@echo "Targets:"
@echo "  venv            - create python venv"
@echo "  deps            - install Python deps"
@echo "  build           - build all Go binaries"
@echo "  fmt             - go fmt"
@echo "  lint            - go vet"
@echo "  run-fastapi     - start FastAPI predictor (ONNX) on :8000"
@echo "  run-predictor   - start gRPC predictor proxy on :7001"
@echo "  run-decider     - start Decider on :7002"
@echo "  invoker         - single decision"
@echo "  sweep           - decision histogram sweep (50)"
@echo "  sweep-policies  - write experiments/results_policies.csv"
@echo "  analyze         - analysis + plots into analysis/"
@echo "  dynamic         - run dynamic replay experiment"
@echo "  clean           - remove bin/"

venv:
python3 -m venv .venv

deps: venv
$(PIP) install --upgrade pip wheel setuptools
$(PIP) install fastapi uvicorn onnxruntime numpy lightgbm scikit-learn pandas matplotlib tabulate grpcio grpcio-tools

build:
go build -o bin/predictor ./services/predict
go build -o bin/decider   ./services/control
go build -o bin/invoker   ./services/invoker
go build -o bin/invoker_sweep ./services/invoker/sweep.go
go build -o bin/ts_check  ./services/invoker/ts_check.go
go build -o bin/fair_sweep ./services/invoker/fair_sweep.go
go build -o bin/sweep_csv ./services/invoker/sweep_csv.go
go build -o bin/sweep_policies ./services/invoker/sweep_policies.go

fmt:
go fmt ./...

lint:
go vet ./...

run-fastapi:
$(PY) -m uvicorn ml.serve_predictor:app --host 127.0.0.1 --port 8000

run-predictor:
./bin/predictor

run-decider:
./bin/decider

invoker:
./bin/invoker

sweep:
./bin/invoker_sweep

sweep-policies:
./bin/sweep_policies

analyze:
$(PY) analysis/analyze_results.py || true
$(PY) analysis/plot_results.py || true
$(PY) analysis/compare_policies.py || true
$(PY) analysis/dynamic_plots.py || true

dynamic:
PYTHONPATH=. $(PY) experiments/replay_dynamic.py

clean:
rm -rf bin/*
