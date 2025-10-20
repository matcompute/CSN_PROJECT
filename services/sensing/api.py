from fastapi import FastAPI, Body
from pydantic import BaseModel
from datetime import datetime
import csv, os

app = FastAPI(title="CSN Sensing Stub")

FEATURES_CSV = os.environ.get("CSN_FEATURES_CSV", "experiments/telemetry.csv")
os.makedirs(os.path.dirname(FEATURES_CSV), exist_ok=True)
if not os.path.exists(FEATURES_CSV):
    with open(FEATURES_CSV, "w", newline="") as f:
        csv.writer(f).writerow(["ts","tenant","app","bw","rtt","loss","dev_cpu","soc","edge_cpu","input_kb","slo_ms","action","lat_mu","lat_var","en_mu","p95_conf"])

class Feature(BaseModel):
    tenant: str
    app: str
    bw: float
    rtt: float
    loss: float
    dev_cpu: float
    soc: float
    edge_cpu: float
    input_kb: float
    slo_ms: float
    action: str
    lat_mu: float | None = None
    lat_var: float | None = None
    en_mu: float | None = None
    p95_conf: float | None = None

@app.post("/ingest")
def ingest(feat: Feature = Body(...)):
    row = [datetime.utcnow().isoformat(), feat.tenant, feat.app, feat.bw, feat.rtt, feat.loss,
           feat.dev_cpu, feat.soc, feat.edge_cpu, feat.input_kb, feat.slo_ms, feat.action,
           feat.lat_mu, feat.lat_var, feat.en_mu, feat.p95_conf]
    with open(FEATURES_CSV, "a", newline="") as f:
        csv.writer(f).writerow(row)
    return {"ok": True}
