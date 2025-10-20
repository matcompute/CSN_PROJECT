#!/usr/bin/env bash
# Usage: ./scripts/retrain_and_roll.sh [reason]
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

STAMP="$(date +%Y%m%d-%H%M%S)"
NEW_DIR="models/v${STAMP}"
mkdir -p "$NEW_DIR"

# 1) (placeholder) copy current models into new version dir
cp models/*.onnx models/*.pkl models/conformal.json "$NEW_DIR/"

# 2) write a tiny manifest for provenance
cat > "$NEW_DIR/manifest.json" <<JSON
{
  "version": "v${STAMP}",
  "created_at": "$(date -u +%FT%TZ)",
  "source": "rollover",
  "reason": "${1-adhoc}",
  "inputs": {
    "telemetry_csv": "experiments/telemetry.csv"
  }
}
JSON

# 3) atomically switch active pointer
ln -sfn "$(pwd)/$NEW_DIR" models/current

echo "[csn] rolled models/current -> $NEW_DIR"
ls -l models | sed -n '1,10p'
