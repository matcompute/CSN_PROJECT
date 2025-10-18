from fastapi import FastAPI
from pydantic import BaseModel
import numpy as np, json
import onnxruntime as ort
from pathlib import Path

app = FastAPI(title="CSN ONNX Predictor (conformal)")

lat_sess = ort.InferenceSession("models/latency.onnx", providers=["CPUExecutionProvider"])
en_sess  = ort.InferenceSession("models/energy.onnx",  providers=["CPUExecutionProvider"])

# load conformal q-hat
_qhat = {"edge:low":8.0,"edge:med":8.0,"edge:high":10.0,"local:med":8.0,"cloud:low":12.0}
conf = Path("models/conformal.json")
if conf.exists():
    data = json.loads(conf.read_text())
    if "qhat" in data:
        _qhat.update({k: float(v) for k,v in data["qhat"].items()})
        print("[conformal] loaded q-hat:", _qhat)

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
    if not a: return "edge","med"
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

    # Tier/Kind effects (same as before)
    tier_lat_mult = {"low": 1.25, "med": 1.00, "high": 0.97}[tier]
    tier_en_mult  = {"low": 0.90, "med": 1.00, "high": 1.40}[tier]

    if kind == "local":
        lat_adj = +3.0; en_mult = 1.25; lat_mult = 1.00; var_l_mult = 0.9
    elif kind == "edge":
        lat_adj = -10.0; en_mult = 0.80; lat_mult = 0.96; var_l_mult = 1.0
    else:
        lat_adj = +45.0; en_mult = 0.70; lat_mult = 1.05; var_l_mult = 1.3

    if kind == "edge":
        load_penalty = max(0.0, (edge_cpu - 0.6)) * 40.0
        lat_adj += load_penalty
        var_l_mult *= (1.0 + max(0.0, edge_cpu - 0.5))

    lat = max(1.0, (lat * lat_mult * tier_lat_mult) + lat_adj)
    en  = max(0.01, en * en_mult * tier_en_mult)

    # variance placeholders (could be learned later)
    var_l = 25.0 * var_l_mult * (1.35 if tier == "high" else 1.0) * (1.25 if tier == "low" else 1.0)
    var_e = 0.02 if tier == "high" else 0.01

    # conformal p95 = mu + q-hat(kind:tier)
    bucket = f"{kind}:{tier}"
    qhat = _qhat.get(bucket, 10.0)
    p95 = lat + qhat

    return PredictOut(
        mu_latency_ms=lat,
        var_latency=var_l,
        mu_energy_j=en,
        var_energy=var_e,
        p95_conformal_ms=p95
    )
