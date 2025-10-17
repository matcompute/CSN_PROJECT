import numpy as np, pandas as pd, os
from sklearn.model_selection import train_test_split
from sklearn.metrics import r2_score
from sklearn.ensemble import GradientBoostingRegressor
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import FloatTensorType

rng = np.random.default_rng(42)

def synth_data(n=8000):
    bw = rng.uniform(2, 120, n)
    rtt = rng.uniform(5, 120, n)
    loss = rng.uniform(0, 0.02, n)
    dcpu = rng.uniform(0.05, 0.95, n)
    ecpu = rng.uniform(0.05, 0.95, n)
    size = rng.uniform(16, 2048, n)
    slo = rng.uniform(60, 240, n)
    X = np.stack([bw, rtt, loss, dcpu, ecpu, size, slo], axis=1).astype(np.float32)

    tx = (size * 8.0 / (bw * 1e3)) * 1e3
    prop = rtt * 0.5
    q = (ecpu**3)*80 + (loss*8000) + rng.normal(0,5,n)
    comp = (size**0.6) * (0.3 + ecpu*0.7) * 0.8
    latency = np.clip(tx + prop + q + comp, 5, None)

    e_cpu = dcpu*0.8 + (size/2048)*0.2
    e_tx  = (size/bw)*0.5 + loss*2.0
    energy = np.clip(e_cpu + e_tx + rng.normal(0,0.05,n), 0.05, None)

    return X, latency.astype(np.float32), energy.astype(np.float32)

def train_and_export():
    X, y_lat, y_en = synth_data()
    Xtr, Xte, ytr_l, yte_l = train_test_split(X, y_lat, test_size=0.2, random_state=1)
    Xtr2, Xte2, ytr_e, yte_e = train_test_split(X, y_en,  test_size=0.2, random_state=1)

    m_lat = GradientBoostingRegressor(n_estimators=300, learning_rate=0.06, max_depth=3, subsample=0.9)
    m_en  = GradientBoostingRegressor(n_estimators=300, learning_rate=0.06, max_depth=3, subsample=0.9)
    m_lat.fit(Xtr, ytr_l); m_en.fit(Xtr2, ytr_e)

    p_l = m_lat.predict(Xte); p_e = m_en.predict(Xte2)
    print(f"R2 latency: {r2_score(yte_l, p_l):.3f} | R2 energy: {r2_score(yte_e, p_e):.3f}")

    os.makedirs("models", exist_ok=True)
    initial_type = [("input", FloatTensorType([None, 7]))]
    onnx_lat = convert_sklearn(m_lat, initial_types=initial_type)
    onnx_en  = convert_sklearn(m_en,  initial_types=initial_type)

    with open("models/latency.onnx", "wb") as f: f.write(onnx_lat.SerializeToString())
    with open("models/energy.onnx",  "wb") as f: f.write(onnx_en.SerializeToString())
    print("Saved: models/latency.onnx , models/energy.onnx")

if __name__ == "__main__":
    train_and_export()
