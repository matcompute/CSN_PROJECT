# Policy comparison

Total rows: **11520**

## Per-policy metrics
| policy         |        n |   viol_rate |   lat_ms |   p95_ms |   energy_j |
|:---------------|---------:|------------:|---------:|---------:|-----------:|
| AlwaysCloudLow | 2304.000 |       1.000 |  200.000 |  240.000 |      0.500 |
| AlwaysEdgeMed  | 2304.000 |       1.000 |  200.000 |  240.000 |      0.500 |
| AlwaysLocal    | 2304.000 |       1.000 |  200.000 |  240.000 |      0.500 |
| CSN            | 2304.000 |       1.000 |  200.000 |  240.000 |      0.500 |
| GreedyLatency  | 2304.000 |       1.000 |  200.000 |  240.000 |      0.500 |

- **Lower is better** for `viol_rate` and `lat_ms`.  
- The Pareto front (approx.) contains points (viol_rate, lat_ms):  
  [[1.0, 200.0], [1.0, 200.0], [1.0, 200.0], [1.0, 200.0], [1.0, 200.0]]
