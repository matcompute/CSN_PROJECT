# CSN Evaluation — Quick Summary

- Total decisions: **2304**
- Overall SLO violation rate: **0.734**

## Per-action breakdown
| chosen_action   |    count |   viol_rate |   latency_ms |   energy_j |
|:----------------|---------:|------------:|-------------:|-----------:|
| edge1:med       | 1356.000 |       0.876 |      644.340 |     28.880 |
| edge1:low       |  454.000 |       0.119 |      174.465 |      4.046 |
| edge1:high      |  372.000 |       0.941 |      281.432 |     10.730 |
| cloud1:low      |   61.000 |       0.984 |      820.181 |     19.170 |
| local:med       |   61.000 |       0.656 |      435.639 |     25.266 |

## Violation vs Bandwidth
| bw_mbps   |   viol_rate |
|:----------|------------:|
| 0-10      |       0.958 |
| 10-20     |       0.778 |
| 20-50     |       0.656 |
| 50-100    |       0.545 |
| 100+      |     nan     |

## Violation vs RTT
| rtt_ms   |   viol_rate |
|:---------|------------:|
| 0-20     |       0.681 |
| 20-50    |       0.715 |
| 50-80    |       0.752 |
| 80-120   |       0.790 |
| 120+     |     nan     |
