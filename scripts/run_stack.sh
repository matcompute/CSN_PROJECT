#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

pkill -f "uvicorn .*services.operator.api" 2>/dev/null || true
pkill -f "uvicorn .*services.sensing.api"  2>/dev/null || true
pkill -f "/bin/predictor$" 2>/dev/null || true
pkill -f "/bin/decider$" 2>/dev/null || true
docker rm -f csn-prom 2>/dev/null || true
sleep 0.3

./scripts/go_next.sh

source .venv/bin/activate
nohup uvicorn services.sensing.api:app --host 0.0.0.0 --port 9104 > /tmp/csn-sensing.log 2>&1 &
deactivate

mkdir -p ops
cat > ops/prometheus.yml <<'YAML'
global:
  scrape_interval: 2s
scrape_configs:
  - job_name: csn-operator
    static_configs:
      - targets: ['host.docker.internal:9103']
        labels: {service: 'operator'}
  - job_name: csn-sensing
    static_configs:
      - targets: ['host.docker.internal:9104']
        labels: {service: 'sensing'}
  - job_name: csn-decider
    static_configs:
      - targets: ['host.docker.internal:9102']
        labels: {service: 'decider'}
YAML

docker run -d --name csn-prom \
  -p 9090:9090 \
  -v "$PWD/ops/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  prom/prometheus:latest \
  --config.file=/etc/prometheus/prometheus.yml

sleep 2
echo "---- PROMETHEUS TARGETS ----"
curl -s "http://127.0.0.1:9090/api/v1/targets?state=active" \
 | grep -E 'csn-(operator|sensing|decider)|\"health\":\"up\"' || true

