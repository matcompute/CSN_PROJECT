from fastapi import FastAPI, Response
from pydantic import BaseModel, Field
from pathlib import Path
from prometheus_client import Counter, Gauge, CONTENT_TYPE_LATEST, generate_latest
from datetime import datetime
import csv

app = FastAPI(title="CSN Sensing API")

# --- metrics ---
INGEST_TOTAL = Counter("csn_sense_ingest_total", "Telemetry rows ingested", ["source"])
LAST_RTT_MS  = Gauge("csn_sense_last_rtt_ms", "Last observed RTT (ms)")
LAST_BW_Mbps = Gauge("csn_sense_last_bw_mbps", "Last observed bandwidth (Mbps)")
LAST_LOSS    = Gauge("csn_sense_last_loss", "Last observed loss rate (0..1)")

OUT_CSV = Path("experiments/telemetry.csv")
OUT_CSV.parent.mkdir(parents=True, exist_ok=True)
HEADER = ["ts","tenant","app","bw_mbps","rtt_ms","loss","device_cpu","edge_cpu",
          "input_kb","slo_p95_ms","action","obs_latency_ms","obs_energy_j"]

class Telemetry(BaseModel):
    tenant: str
    app: str
    bw_mbps: float
    rtt_ms: float
    loss: float = Field(ge=0.0, le=1.0)
    device_cpu: float = Field(ge=0.0, le=1.0)
    edge_cpu: float = Field(ge=0.0, le=1.0)
    input_kb: float
    slo_p95_ms: float
    action: str
    obs_latency_ms: float
    obs_energy_j: float
    ts: int | None = None

@app.post("/ingest")
def ingest(row: Telemetry):
    ts = row.ts if row.ts is not None else int(datetime.utcnow().timestamp())
    LAST_RTT_MS.set(row.rtt_ms)
    LAST_BW_Mbps.set(row.bw_mbps)
    LAST_LOSS.set(row.loss)
    INGEST_TOTAL.labels(source="api").inc()

    write_header = not OUT_CSV.exists()
    with OUT_CSV.open("a", newline="") as f:
        w = csv.writer(f)
        if write_header:
            w.writerow(HEADER)
        w.writerow([
            ts, row.tenant, row.app, row.bw_mbps, row.rtt_ms, row.loss,
            row.device_cpu, row.edge_cpu, row.input_kb, row.slo_p95_ms,
            row.action, row.obs_latency_ms, row.obs_energy_j
        ])
    return {"ok": True, "written": str(OUT_CSV)}

@app.get("/metrics")
def metrics():
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)
