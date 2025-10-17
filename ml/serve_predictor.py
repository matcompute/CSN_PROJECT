from fastapi import FastAPI
from pydantic import BaseModel
import numpy as np
import onnxruntime as ort

app = FastAPI(title="CSN ONNX Predictor")

lat_sess = ort.InferenceSession("models/latency.onnx", providers=["CPUExecutionProvider"])
en_sess  = ort.InferenceSession("models/energy.onnx",  providers=["CPUExecutionProvider"])

class PredictIn(BaseModel):
    features: list[float]   # [bw, rtt, loss, device_cpu, edge_cpu, input_kb, slo_p95_ms]
    action: str | None = None

class PredictOut(BaseModel):
    mu_latency_ms: float
    var_latency: float
    mu_energy_j: float
    var_energy: float
    p95_conformal_ms: float

def parse_action(a: str | None):
    if not a: return "edge", "med"
    parts = a.split(":")
    kind = parts[0]
    tier = parts[1] if len(parts) > 1 else "med"
    if kind.startswith("edge"): kind = "edge"
    if kind.startswith("cloud"): kind = "cloud"
    if kind not in ("local","edge","cloud"): kind = "edge"
    if tier not in ("low","med","high"): tier = "med"
    return kind, tier

@app.post("/predict", response_model=PredictOut)
def predict(inp: PredictIn):
    x = np.asarray(inp.features, dtype=np.float32).reshape(1, 7)
    lat = float(lat_sess.run(None, {"input": x})[0].ravel()[0])
    en  = float(en_sess.run(None,  {"input": x})[0].ravel()[0])

    bw, rtt, loss, device_cpu, edge_cpu, size, slo = map(float, inp.features)
    kind, tier = parse_action(inp.action)

    # Tier multipliers: make HIGH only slightly faster but much more energy hungry
    tier_lat_mult = {"low": 1.25, "med": 1.00, "high": 0.97}[tier]
    tier_en_mult  = {"low": 0.90, "med": 1.00, "high": 1.40}[tier]  # big energy cost for high

    # Kind adjustments: cloud adds WAN latency but reduces device energy; local saves WAN but burns CPU
    if kind == "local":
        lat_adj = +3.0
        en_mult = 1.25  # device pays
        lat_mult = 1.00
        var_l_mult = 0.9
    elif kind == "edge":
        lat_adj = -10.0
        en_mult = 0.80
        lat_mult = 0.96
        var_l_mult = 1.0
    else:  # cloud
        lat_adj = +45.0  # WAN + queuing
        en_mult = 0.70
        lat_mult = 1.05
        var_l_mult = 1.3  # riskier tails

    # Edge load effect: when edge_cpu is high, edge tiers have diminishing returns
    if kind == "edge":
        # more loaded edge => less benefit from higher tier
        load_penalty = max(0.0, (edge_cpu - 0.6)) * 40.0  # up to +16ms around 1.0
        lat_adj += load_penalty
        # and variance increases with load
        var_l_mult *= (1.0 + max(0.0, edge_cpu - 0.5))

    # Apply adjustments
    lat = max(1.0, (lat * lat_mult * tier_lat_mult) + lat_adj)
    en  = max(0.01, en * en_mult * tier_en_mult)

    # Variance: depend on kind/tier; HIGH is riskier due to resource contention
    base_var_l = 25.0 * var_l_mult
    if tier == "low":
        base_var_l *= 1.25
    elif tier == "high":
        base_var_l *= 1.35  # make high-tier tails riskier
    var_l = base_var_l
    var_e = 0.02 if tier == "high" else 0.01

    p95 = lat + 1.645 * (var_l ** 0.5)
    return PredictOut(
        mu_latency_ms=lat,
        var_latency=var_l,
        mu_energy_j=en,
        var_energy=var_e,
        p95_conformal_ms=p95
    )
