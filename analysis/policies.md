# Policy comparison

Total rows: **11520**

## Per-policy metrics
| policy         |        n |   viol_rate |   lat_ms |   p95_ms |   energy_j |
|:---------------|---------:|------------:|---------:|---------:|-----------:|
| GreedyLatency  | 2304.000 |       0.718 |  464.468 |  474.763 |     28.270 |
| AlwaysEdgeMed  | 2304.000 |       0.726 |  479.017 |  487.878 |     20.193 |
| CSN            | 2304.000 |       0.735 |  482.474 |  492.445 |     26.939 |
| AlwaysLocal    | 2304.000 |       0.760 |  508.203 |  516.006 |     31.552 |
| AlwaysCloudLow | 2304.000 |       0.912 |  708.079 |  718.564 |     15.902 |

- **Lower is better** for `viol_rate` and `lat_ms`.  
- The Pareto front (approx.) contains points (viol_rate, lat_ms):  
  [[0.718, 464.468]]
