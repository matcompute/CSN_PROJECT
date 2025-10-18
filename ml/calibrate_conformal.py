import json, os, numpy as np
from pathlib import Path
import onnxruntime as ort

rng = np.random.default_rng(7)

def parse_action(a):
    if not a: return "edge","med"
    parts = a.split(":")
    kind = parts[0]
    tier = parts[1] if len(parts) > 1 else "med"
    if kind.startswith("edge"):  kind = "edge"
    if kind.startswith("cloud"): kind = "cloud"
    if kind not in ("local","edge","cloud"): kind = "edge"
    if tier not in ("low","med","high"): tier = "med"
    return kind, tier

def apply_action_adjustments(base_lat, base_en, features, action):
    # must match ml/serve_predictor.py (keep in sync!)
    bw, rtt, loss, device_cpu, edge_cpu, size, slo = map(float, features)
    kind, tier = parse_action(action)
    tier_lat_mult = {"low": 1.25, "med": 1.00, "high": 0.97}[tier]
    tier_en_mult  = {"low": 0.90, "med": 1.00, "high": 1.40}[tier]
    if kind == "local":
        lat_adj = +3.0;  en_mult = 1.25; lat_mult = 1.00
    elif kind == "edge":
        lat_adj = -10.0; en_mult = 0.80; lat_mult = 0.96
    else:
        lat_adj = +45.0; en_mult = 0.70; lat_mult = 1.05
    if kind == "edge":
        load_penalty = max(0.0, (float(features[4]) - 0.6)) * 40.0
        lat_adj += load_penalty
    lat = max(1.0, (base_lat * lat_mult * tier_lat_mult) + lat_adj)
    en  = max(0.01, base_en * en_mult * tier_en_mult)
    return lat, en

def synth_features(n):
    bw   = rng.uniform(2, 120, n)
    rtt  = rng.uniform(5, 120, n)
    loss = rng.uniform(0, 0.02, n)
    dcpu = rng.uniform(0.05, 0.95, n)
    ecpu = rng.uniform(0.05, 0.95, n)
    size = rng.uniform(16, 2048, n)
    slo  = rng.uniform(60, 240, n)
    return np.stack([bw, rtt, loss, dcpu, ecpu, size, slo], axis=1).astype(np.float32)

def true_latency(features, action):
    bw, rtt, loss, dcpu, ecpu, size, slo = map(float, features)
    tx = (size * 8.0 / (bw * 1e3)) * 1e3
    prop = rtt * 0.5
    q = (ecpu**3)*80 + (loss*8000)
    comp = (size**0.6) * (0.3 + ecpu*0.7) * 0.8
    base = max(5.0, tx + prop + q + comp)
    lat, _ = apply_action_adjustments(base, 1.0, features, action)
    return lat

def main():
    lat_sess = ort.InferenceSession("models/latency.onnx", providers=["CPUExecutionProvider"])
    en_sess  = ort.InferenceSession("models/energy.onnx",  providers=["CPUExecutionProvider"])
    actions = ["local:med","edge1:low","edge1:med","edge1:high","cloud1:low"]
    buckets = {("local","med"):[],("edge","low"):[],("edge","med"):[],("edge","high"):[],("cloud","low"):[]}
    X = synth_features(8000)
    base_lat = lat_sess.run(None, {"input": X})[0].ravel()
    base_en  = en_sess.run(None,  {"input": X})[0].ravel()

    for a in actions:
        kind,tier = parse_action(a)
        # predicted mean with action adjustments
        pred_lat = np.array([apply_action_adjustments(base_lat[i], base_en[i], X[i], a)[0] for i in range(len(X))], dtype=np.float32)
        # synthetic “true” latency with action
        true_lat = np.array([true_latency(X[i], a) for i in range(len(X))], dtype=np.float32)
        resid = np.maximum(true_lat - pred_lat, 0.0)  # positive errors
        buckets[(kind,tier)].extend(resid.tolist())

    qhat = {}
    alpha = 0.95
    for k, vals in buckets.items():
        arr = np.array(vals, dtype=np.float32)
        q = float(np.quantile(arr, alpha, method="higher")) if len(arr) else 0.0
        qhat[f"{k[0]}:{k[1]}"] = q

    os.makedirs("models", exist_ok=True)
    outp = Path("models/conformal.json")
    outp.write_text(json.dumps({"alpha": alpha, "qhat": qhat}, indent=2))
    print("Saved conformal quantiles to", outp)
    for k,v in qhat.items():
        print(f"{k:10s} q̂={v:.2f} ms")

if __name__ == "__main__":
    main()
