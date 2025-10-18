import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from pathlib import Path

DATA = Path("experiments/results.csv")
OUT = Path("analysis")
OUT.mkdir(parents=True, exist_ok=True)

df = pd.read_csv(DATA)

# --- 1) CDF of predicted latency (overall + by top actions) ---
plt.figure()
# overall
x = np.sort(df["mu_latency_ms"].values)
y = np.linspace(0, 1, len(x), endpoint=True)
plt.plot(x, y, label="overall")
# top actions
for a in ["local:med", "edge1:low", "edge1:med", "edge1:high", "cloud1:low"]:
    sub = df[df["chosen_action"] == a]["mu_latency_ms"].values
    if len(sub) == 0: 
        continue
    xs = np.sort(sub)
    ys = np.linspace(0, 1, len(xs), endpoint=True)
    plt.plot(xs, ys, label=a)
plt.xlabel("Predicted latency (ms)")
plt.ylabel("CDF")
plt.title("Latency CDF (predicted)")
plt.legend()
plt.grid(True, alpha=0.3)
lat_cdf_path = OUT / "latency_cdf.png"
plt.savefig(lat_cdf_path, bbox_inches="tight", dpi=140)
plt.close()

# --- 2) Per-action: selection share and violation rate ---
agg = (df.groupby("chosen_action")
         .agg(count=("chosen_action","size"),
              viol_rate=("slo_viol","mean"))
         .sort_values("count", ascending=False))
actions = agg.index.tolist()
counts = agg["count"].values
shares = counts / counts.sum()
viol = agg["viol_rate"].values

# bar chart 1: selection share
plt.figure()
plt.bar(actions, shares)
plt.ylabel("Selection share")
plt.title("Action selection share")
plt.xticks(rotation=30, ha="right")
plt.grid(axis="y", alpha=0.3)
share_path = OUT / "action_share.png"
plt.savefig(share_path, bbox_inches="tight", dpi=140)
plt.close()

# bar chart 2: violation rate
plt.figure()
plt.bar(actions, viol)
plt.ylabel("SLO violation rate")
plt.title("Violation rate by action")
plt.xticks(rotation=30, ha="right")
plt.grid(axis="y", alpha=0.3)
viol_path = OUT / "action_violation.png"
plt.savefig(viol_path, bbox_inches="tight", dpi=140)
plt.close()

# append to summary.md
md = OUT / "summary.md"
append = f"""

## Figures
- Latency CDF: `{lat_cdf_path}`
- Action selection share: `{share_path}`
- Violation rate by action: `{viol_path}`
"""
with md.open("a") as f:
    f.write(append)

print("Wrote:", lat_cdf_path, share_path, viol_path)
