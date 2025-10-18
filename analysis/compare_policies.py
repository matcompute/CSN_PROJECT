import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from pathlib import Path

DATA = Path("experiments/results_policies.csv")
OUT  = Path("analysis"); OUT.mkdir(exist_ok=True, parents=True)

if not DATA.exists():
    raise SystemExit("experiments/results_policies.csv not found. Run ./bin/sweep_policies first.")

df = pd.read_csv(DATA)

# core metrics per policy
per = (df.groupby("policy")
         .agg(
             n=("policy","size"),
             viol_rate=("slo_viol","mean"),
             lat_ms=("mu_latency_ms","mean"),
             p95_ms=("p95_conformal_ms","mean"),
             energy_j=("mu_energy_j","mean")
         )
         .sort_values(["viol_rate","lat_ms"], ascending=[True, True])
      )

# Pareto-ish identification: minimize (viol_rate, lat_ms)
def pareto_front(points):
    pts = points.copy()
    front = []
    while len(pts):
        best = pts[:,0].argmin()  # min violation
        cand = pts[best]
        # keep any point not dominated by 'cand'
        nondom = []
        for i,p in enumerate(pts):
            if i==best: continue
            # p is dominated if cand has <= viol and <= lat and one <
            dominated = (cand[0] <= p[0] and cand[1] <= p[1]) and (cand[0] < p[0] or cand[1] < p[1])
            if not dominated:
                nondom.append(p)
        front.append(cand)
        pts = np.array(nondom) if nondom else np.empty((0,2))
    return np.array(front)

pts = per[["viol_rate","lat_ms"]].to_numpy()
pf  = pareto_front(pts)
policies = per.index.tolist()

# write markdown summary
md = OUT / "policies.md"
md.write_text(
f"""# Policy comparison

Total rows: **{len(df)}**

## Per-policy metrics
{per.to_markdown(floatfmt=".3f")}

- **Lower is better** for `viol_rate` and `lat_ms`.  
- The Pareto front (approx.) contains points (viol_rate, lat_ms):  
  {pf.round(3).tolist()}
"""
)

print("Per-policy metrics:")
print(per.to_string(float_format=lambda x: f"{x:.3f}"))
print("\nPareto front (viol_rate, lat_ms):", np.round(pf,3).tolist())

# ---- Plots ----

# 1) Bar: violation rate
plt.figure()
per["viol_rate"].plot(kind="bar")
plt.ylabel("SLO violation rate")
plt.title("SLO violation rate by policy")
plt.grid(axis="y", alpha=0.3)
plt.tight_layout()
plt.savefig(OUT / "policies_violation.png", dpi=140)
plt.close()

# 2) Bar: mean latency
plt.figure()
per["lat_ms"].plot(kind="bar")
plt.ylabel("Mean predicted latency (ms)")
plt.title("Mean latency by policy")
plt.grid(axis="y", alpha=0.3)
plt.tight_layout()
plt.savefig(OUT / "policies_latency.png", dpi=140)
plt.close()

# 3) Scatter: latency vs violation (label points)
plt.figure()
x = per["viol_rate"].values
y = per["lat_ms"].values
plt.scatter(x, y)
for i, name in enumerate(per.index):
    plt.annotate(name, (x[i], y[i]), xytext=(3,3), textcoords="offset points", fontsize=8)
plt.xlabel("SLO violation rate")
plt.ylabel("Mean predicted latency (ms)")
plt.title("Latency vs SLO violation (policies)")
plt.grid(True, alpha=0.3)
plt.tight_layout()
plt.savefig(OUT / "policies_scatter.png", dpi=140)
plt.close()

print("Wrote:",
      OUT / "policies.md",
      OUT / "policies_violation.png",
      OUT / "policies_latency.png",
      OUT / "policies_scatter.png")
