package main

import (
"bytes"
"encoding/json"
"net/http"
"os"
"time"

pb "github.com/mulat/csn/proto"
)

var senseURL = func() string {
if v := os.Getenv("CSN_SENSE_URL"); v != "" { return v }
return "http://127.0.0.1:9105/ingest"
}()

type sensePayload struct {
Tenant  string   `json:"tenant"`
App     string   `json:"app"`
Bw      float64  `json:"bw"`
Rtt     float64  `json:"rtt"`
Loss    float64  `json:"loss"`
DevCPU  float64  `json:"dev_cpu"`
Soc     float64  `json:"soc"`
EdgeCPU float64  `json:"edge_cpu"`
InputKB float64  `json:"input_kb"`
SloMS   float64  `json:"slo_ms"`
Action  string   `json:"action"`
LatMu   *float64 `json:"lat_mu,omitempty"`
LatVar  *float64 `json:"lat_var,omitempty"`
EnMu    *float64 `json:"en_mu,omitempty"`
P95Conf *float64 `json:"p95_conf,omitempty"`
}

func postSense(c *pb.Context, action string) {
if c == nil || action == "" { return }
body := sensePayload{
Tenant:  c.GetTenantId(),
App:     c.GetAppId(),
Bw:      c.GetBwMbps(),
Rtt:     c.GetRttMs(),
Loss:    c.GetLoss(),
DevCPU:  c.GetDeviceCpu(),
Soc:     c.GetBatterySoc(),
EdgeCPU: c.GetEdgeCpu(),
InputKB: c.GetInputKb(),
SloMS:   c.GetSloP95Ms(),
Action:  action,
}
buf, _ := json.Marshal(body)
req, _ := http.NewRequest("POST", senseURL, bytes.NewReader(buf))
req.Header.Set("Content-Type", "application/json")
client := &http.Client{ Timeout: 300 * time.Millisecond }
go client.Do(req) // fire-and-forget
}
