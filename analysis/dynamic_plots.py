import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from pathlib import Path

DATA = Path("experiments/dynamic.csv")
OUT  = Path("analysis"); OUT.mkdir(parents=True, exist_ok=True)

df = pd.read_csv(DATA)
# coerce numeric cols (they're strings in CSV)
numcols = ["t","bw_mbps","rtt_ms","loss","edge_cpu","input_kb","slo_p95_ms",
           "mu_latency_ms","mu_energy_j","p95_conformal_ms","slo_viol"]
for c in numcols:
    df[c] = pd.to_numeric(df[c], errors="coerce")

# ----- 1) p95 vs SLO over time with phases -----
plt.figure()
plt.plot(df["t"], df["p95_conformal_ms"], label="p95_conformal")
plt.plot(df["t"], df["slo_p95_ms"], label="SLO")
# shade phases
phases = df["phase"].values
tvals = df["t"].values
last = 0
for i in range(1, len(df)):
    if phases[i] != phases[i-1]:
        plt.axvspan(tvals[last], tvals[i-1], alpha=0.1)
        last = i
# close last span
plt.axvspan(tvals[last], tvals[-1], alpha=0.1)
plt.xlabel("time (steps)")
plt.ylabel("latency (ms)")
plt.title("p95 vs SLO over time")
plt.grid(True, alpha=0.3)
plt.legend()
p95_path = OUT / "dyn_p95_vs_slo.png"
plt.savefig(p95_path, bbox_inches="tight", dpi=140)
plt.close()

# ----- 2) Rolling SLO violation rate (window=20) -----
w = 20
roll = df["slo_viol"].rolling(w, min_periods=1).mean()
plt.figure()
plt.plot(df["t"], roll)
plt.axhline(0.10, linestyle="--")  # target 10%
plt.xlabel("time (steps)")
plt.ylabel(f"rolling violation rate (window={w})")
plt.title("SLO violation rate over time")
plt.grid(True, alpha=0.3)
viol_path = OUT / "dyn_rolling_viol.png"
plt.savefig(viol_path, bbox_inches="tight", dpi=140)
plt.close()

# ----- 3) Action choice over time (encoded as integers) -----
# map actions to ints for a clean line plot
actions = df["chosen_action"].astype(str).values
uniq = {a:i for i,a in enumerate(sorted(df["chosen_action"].unique()))}
enc = [uniq[a] for a in actions]
plt.figure()
plt.plot(df["t"], enc, drawstyle="steps-post")
plt.yticks(list(uniq.values()), list(uniq.keys()), rotation=0)
plt.xlabel("time (steps)")
plt.ylabel("chosen action")
plt.title("Action trajectory")
plt.grid(True, alpha=0.3)
act_path = OUT / "dyn_actions.png"
plt.savefig(act_path, bbox_inches="tight", dpi=140)
plt.close()

# append to summary
md = OUT / "summary.md"
with md.open("a") as f:
    f.write(f"""

## Dynamic scenario figures
- p95 vs SLO over time: `{p95_path}`
- Rolling SLO violation rate: `{viol_path}`
- Action trajectory: `{act_path}`
""")
print("Wrote:", p95_path, viol_path, act_path)
