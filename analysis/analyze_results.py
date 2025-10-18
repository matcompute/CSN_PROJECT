import pandas as pd
from pathlib import Path

csv = Path("experiments/results.csv")
if not csv.exists():
    raise SystemExit("experiments/results.csv not found. Run ./bin/sweep_csv first.")

df = pd.read_csv(csv)

# core metrics
N = len(df)
viol_rate = df["slo_viol"].mean()

# per-action breakdown
per_action = (
    df.groupby("chosen_action")
      .agg(count=("chosen_action","size"),
           viol_rate=("slo_viol","mean"),
           latency_ms=("mu_latency_ms","mean"),
           energy_j=("mu_energy_j","mean"),
          )
      .sort_values("count", ascending=False)
)

# by bandwidth buckets
bw_bins = pd.cut(df["bw_mbps"], bins=[0,10,20,50,100,1e9], right=True,
                 labels=["0-10","10-20","20-50","50-100","100+"])
by_bw = df.groupby(bw_bins)["slo_viol"].mean().rename("viol_rate").to_frame()

# by RTT buckets
rtt_bins = pd.cut(df["rtt_ms"], bins=[0,20,50,80,120,1e9], right=True,
                  labels=["0-20","20-50","50-80","80-120","120+"])
by_rtt = df.groupby(rtt_bins)["slo_viol"].mean().rename("viol_rate").to_frame()

# write a tiny markdown summary for the paper
md = Path("analysis/summary.md")
md.write_text(f"""# CSN Evaluation — Quick Summary

- Total decisions: **{N}**
- Overall SLO violation rate: **{viol_rate:.3f}**

## Per-action breakdown
{per_action.to_markdown(floatfmt=".3f")}

## Violation vs Bandwidth
{by_bw.to_markdown(floatfmt=".3f")}

## Violation vs RTT
{by_rtt.to_markdown(floatfmt=".3f")}
""")

print(f"Total decisions: {N}")
print(f"Overall SLO violation rate: {viol_rate:.3f}\n")
print("Per-action breakdown:")
print(per_action.to_string(float_format=lambda x: f'{x:.3f}'))
print("\nViolation vs Bandwidth:")
print(by_bw.to_string(float_format=lambda x: f'{x:.3f}'))
print("\nViolation vs RTT:")
print(by_rtt.to_string(float_format=lambda x: f'{x:.3f}'))

