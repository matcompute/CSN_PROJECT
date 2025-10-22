import re, sys
from pathlib import Path
import pandas as pd
import matplotlib.pyplot as plt

# Find latest run folder
runs = Path("experiments/runs")
cands = sorted([d for d in runs.glob("*") if d.is_dir()])
if not cands:
    print("No runs found in experiments/runs")
    sys.exit(0)

run = cands[-1]  # most recent
print("Using run folder:", run)

def parse_actions(path: Path):
    rows = []
    for line in path.read_text().splitlines():
        if line.startswith("Chosen action:"):
            m = re.search(r"Chosen action:\s+([a-zA-Z0-9]+):([a-zA-Z0-9]+)", line)
            if m:
                kind_raw, tier = m.group(1), m.group(2)
                if kind_raw.startswith("edge"):
                    kind = "edge"
                elif kind_raw.startswith("cloud"):
                    kind = "cloud"
                elif kind_raw.startswith("local"):
                    kind = "local"
                else:
                    kind = "other"
                rows.append({"phase": path.stem, "kind": kind, "tier": tier})
    return rows

# Parse all phase files
all_rows = []
for f in sorted(run.glob("phase*.out")):
    all_rows.extend(parse_actions(f))

if not all_rows:
    print("No actions found in", run)
    sys.exit(0)

df = pd.DataFrame(all_rows)

# ✅ Robust handling: ensure columns edge/cloud/local exist even if missing
counts = (
    df.groupby(["phase","kind"]).size()
      .unstack(fill_value=0)
      .reindex(columns=["edge","cloud","local"], fill_value=0)
)
counts["total"] = counts.sum(axis=1)
counts.to_csv(run / "summary_counts.csv")
print("Wrote", run / "summary_counts.csv")

# Plot 1: counts
plt.figure(figsize=(8,5))
x = range(len(counts))
edge = counts["edge"].values
cloud = counts["cloud"].values
local = counts["local"].values
plt.bar(x, edge, label="edge")
plt.bar(x, cloud, bottom=edge, label="cloud")
plt.bar(x, local, bottom=edge+cloud, label="local")
plt.xticks(list(x), counts.index)
plt.ylabel("Count")
plt.title("Action mix by phase")
plt.legend()
plt.tight_layout()
p1 = run / "action_mix_by_phase.png"
plt.savefig(p1, dpi=150)
print("Wrote", p1)

# Plot 2: proportions
prop = counts[["edge","cloud","local"]].div(counts["total"], axis=0)
plt.figure(figsize=(8,5))
x = range(len(prop))
plt.bar(x, prop["edge"].values, label="edge")
plt.bar(x, prop["cloud"].values, bottom=prop["edge"].values, label="cloud")
plt.bar(x, prop["local"].values, bottom=(prop["edge"]+prop["cloud"]).values, label="local")
plt.xticks(list(x), prop.index)
plt.ylabel("Proportion")
plt.ylim(0,1)
plt.title("Action mix proportion by phase")
plt.legend()
plt.tight_layout()
p2 = run / "action_mix_proportion.png"
plt.savefig(p2, dpi=150)
print("Wrote", p2)
