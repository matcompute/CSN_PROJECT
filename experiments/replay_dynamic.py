import csv, time, math, random
from pathlib import Path
import grpc
import proto.csn_pb2 as csn
import proto.csn_pb2_grpc as csn_grpc

OUT = Path("experiments/dynamic.csv")
OUT.parent.mkdir(parents=True, exist_ok=True)

FEASIBLE = ["local:med","edge1:low","edge1:med","edge1:high","cloud1:low"]

def phase_profile(phase, t):
    # returns (bw, rtt, loss, edge_cpu, input_kb, slo)
    if phase == "good":
        return (80+10*math.sin(t/15), 15+3*math.sin(t/11), 0.0005, 0.35, 256, 160)
    if phase == "congested":
        return (15+5*math.sin(t/9), 80+15*math.sin(t/7), 0.008, 0.85, 512, 160)
    if phase == "lossy":
        return (25+8*math.sin(t/13), 55+12*math.sin(t/10), 0.015, 0.60, 1024, 160)
    if phase == "recovery":
        return (60+15*math.sin(t/14), 25+5*math.sin(t/12), 0.002, 0.45, 384, 160)
    return (50, 40, 0.005, 0.5, 512, 160)

def main():
    # connect services
    dec_ch = grpc.insecure_channel("127.0.0.1:7002")
    dec = csn_grpc.DeciderStub(dec_ch)

    pred_ch = grpc.insecure_channel("127.0.0.1:7001")
    predictor = csn_grpc.PredictorStub(pred_ch)

    phases = [("good",60), ("congested",60), ("lossy",60), ("recovery",60)]
    t = 0
    with OUT.open("w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["t","phase","bw_mbps","rtt_ms","loss","edge_cpu","input_kb","slo_p95_ms",
                    "chosen_action","mu_latency_ms","mu_energy_j","p95_conformal_ms","slo_viol"])
        for ph, steps in phases:
            for _ in range(steps):
                bw, rtt, loss, ecpu, size, slo = phase_profile(ph, t)
                ctx = csn.Context(
                    tenant_id="tenantA", app_id="app1",
                    bw_mbps=float(bw), rtt_ms=float(rtt), loss=float(loss),
                    device_cpu=0.5, battery_soc=0.7, edge_cpu=float(ecpu),
                    input_kb=float(size), slo_p95_ms=float(slo)
                )
                try:
                    resp = dec.Decide(csn.DecideRequest(ctx=ctx, feasible_actions=FEASIBLE), timeout=1.2)
                except Exception as e:
                    print("decide error:", e)
                    time.sleep(0.05)
                    t += 1
                    continue

                try:
                    pred = predictor.Predict(csn.PredictRequest(ctx=ctx, action=resp.chosen_action), timeout=1.2)
                except Exception as e:
                    print("predict error:", e)
                    time.sleep(0.05)
                    t += 1
                    continue

                viol = 1 if pred.p95_conformal_ms > ctx.slo_p95_ms + 1e-9 else 0
                w.writerow([t, ph, f"{bw:.3f}", f"{rtt:.3f}", f"{loss:.5f}", f"{ecpu:.3f}",
                            f"{size:.1f}", f"{slo:.1f}", resp.chosen_action,
                            f"{pred.mu_latency_ms:.3f}", f"{pred.mu_energy_j:.5f}",
                            f"{pred.p95_conformal_ms:.3f}", viol])
                t += 1
                time.sleep(0.02)  # gentle pacing
    print(f"Wrote {t} rows to {OUT}")
if __name__ == "__main__":
    main()
