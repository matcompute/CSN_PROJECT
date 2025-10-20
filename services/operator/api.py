from fastapi import FastAPI, Response
from pydantic import BaseModel
from prometheus_client import Counter, Gauge, CONTENT_TYPE_LATEST, generate_latest

app = FastAPI(title="CSN Operator API")

# --- metrics ---
OPS_TOTAL = Counter("csn_ops_requests_total", "Operator API requests", ["endpoint"])
EDGES_UP  = Gauge("csn_edges_up", "Number of simulated edge nodes up")

# in-memory state (demo)
edges = set()

class EdgeReq(BaseModel):
    name: str

class EstimateReq(BaseModel):
    action: str
    bw_mbps: float
    rtt_ms: float
    input_kb: float
    slo_p95_ms: float

@app.post("/edge")
def add_edge(req: EdgeReq):
    OPS_TOTAL.labels(endpoint="/edge POST").inc()
    edges.add(req.name)
    EDGES_UP.set(len(edges))
    return {"ok": True, "edges": sorted(list(edges))}

@app.post("/estimate")
def estimate(req: EstimateReq):
    OPS_TOTAL.labels(endpoint="/estimate").inc()
    # dummy values; wire to predictor if needed
    return {
        "action": req.action,
        "mu_latency_ms": 92.3662109375,
        "mu_energy_j": 2.467411804199219,
        "p95_conformal_ms": 122.9664306640625,
        "slo_violation": 0,
    }

@app.get("/metrics")
def metrics():
    data = generate_latest()
    return Response(content=data, media_type=CONTENT_TYPE_LATEST)

@app.post("/drain")
def drain(req: EdgeReq):
    OPS_TOTAL.labels(endpoint="/drain POST").inc()
    edges.discard(req.name)
    EDGES_UP.set(len(edges))
    return {"ok": True, "edges": sorted(list(edges))}
