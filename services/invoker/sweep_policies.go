package main

import (
"context"
"encoding/csv"
"fmt"
"log"
"os"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

type pred struct {
action string
lat float64
energy float64
p95 float64
}

func main() {
// connect decider & predictor
decConn, err := grpc.Dial("127.0.0.1:7002", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil { log.Fatalf("connect decider: %v", err) }
defer decConn.Close()
dec := pb.NewDeciderClient(decConn)

predConn, err := grpc.Dial("127.0.0.1:7001", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil { log.Fatalf("connect predictor: %v", err) }
defer predConn.Close()
predictor := pb.NewPredictorClient(predConn)

feasible := []string{"local:med","edge1:low","edge1:med","edge1:high","cloud1:low"}

// CSV
if err := os.MkdirAll("experiments", 0o755); err != nil { log.Fatal(err) }
f, err := os.Create("experiments/results_policies.csv")
if err != nil { log.Fatal(err) }
defer f.Close()
w := csv.NewWriter(f)
defer w.Flush()

w.Write([]string{
"policy","tenant","app","bw_mbps","rtt_ms","loss","device_cpu","edge_cpu","input_kb","slo_p95_ms",
"chosen_action","mu_latency_ms","mu_energy_j","p95_conformal_ms","slo_viol","ts",
})

// same grid as before (can tune later)
bwList   := []float64{5, 20, 50, 100}
rttList  := []float64{10, 40, 80, 120}
lossList := []float64{0.0, 0.003, 0.01}
ecpuList := []float64{0.2, 0.6, 0.9}
sizeList := []float64{64, 256, 1024, 1536}
sloList  := []float64{100, 140, 180, 220}

total := 0
ctxTimeout := 2 * time.Second

for _, bw := range bwList {
for _, rtt := range rttList {
for _, loss := range lossList {
for _, ecpu := range ecpuList {
for _, size := range sizeList {
for _, slo := range sloList {
ctx := &pb.Context{
TenantId:   "tenantA",
AppId:      "app1",
BwMbps:     bw,
RttMs:      rtt,
Loss:       loss,
DeviceCpu:  0.5,
BatterySoc: 0.7,
EdgeCpu:    ecpu,
InputKb:    size,
SloP95Ms:   slo,
}

// pre-compute predictions for all actions
cache := make(map[string]pred, len(feasible))
for _, a := range feasible {
cctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
p, err := predictor.Predict(cctx, &pb.PredictRequest{Ctx: ctx, Action: a})
cancel()
if err != nil {
log.Printf("predict err: %v", err)
continue
}
cache[a] = pred{action:a, lat: float64(p.MuLatencyMs), energy: float64(p.MuEnergyJ), p95: float64(p.P95ConformalMs)}
}
if len(cache) == 0 { continue }

// policy 1: CSN (Decider)
{
cctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
resp, err := dec.Decide(cctx, &pb.DecideRequest{Ctx: ctx, FeasibleActions: feasible})
cancel()
if err == nil {
p := cache[resp.ChosenAction]
viol := 0; if p.p95 > float64(ctx.SloP95Ms) { viol = 1 }
w.Write([]string{
"CSN", ctx.TenantId, ctx.AppId,
fmt.Sprintf("%.3f", bw),
fmt.Sprintf("%.3f", rtt),
fmt.Sprintf("%.5f", loss),
fmt.Sprintf("%.3f", ctx.DeviceCpu),
fmt.Sprintf("%.3f", ecpu),
fmt.Sprintf("%.1f", size),
fmt.Sprintf("%.1f", slo),
resp.ChosenAction,
fmt.Sprintf("%.3f", p.lat),
fmt.Sprintf("%.5f", p.energy),
fmt.Sprintf("%.3f", p.p95),
fmt.Sprintf("%d", viol),
fmt.Sprintf("%d", time.Now().Unix()),
})
total++
}
}

// policy 2: GreedyLatency (min predicted latency)
{
bestA := ""; bestLat := 1e18
for _, a := range feasible {
p, ok := cache[a]; if !ok { continue }
if p.lat < bestLat { bestLat = p.lat; bestA = a }
}
if bestA != "" {
p := cache[bestA]
viol := 0; if p.p95 > float64(ctx.SloP95Ms) { viol = 1 }
w.Write([]string{
"GreedyLatency", ctx.TenantId, ctx.AppId,
fmt.Sprintf("%.3f", bw),
fmt.Sprintf("%.3f", rtt),
fmt.Sprintf("%.5f", loss),
fmt.Sprintf("%.3f", ctx.DeviceCpu),
fmt.Sprintf("%.3f", ecpu),
fmt.Sprintf("%.1f", size),
fmt.Sprintf("%.1f", slo),
bestA,
fmt.Sprintf("%.3f", p.lat),
fmt.Sprintf("%.5f", p.energy),
fmt.Sprintf("%.3f", p.p95),
fmt.Sprintf("%d", viol),
fmt.Sprintf("%d", time.Now().Unix()),
})
total++
}
}

// policy 3: AlwaysLocal
{
a := "local:med"
if p, ok := cache[a]; ok {
viol := 0; if p.p95 > float64(ctx.SloP95Ms) { viol = 1 }
w.Write([]string{
"AlwaysLocal", ctx.TenantId, ctx.AppId,
fmt.Sprintf("%.3f", bw),
fmt.Sprintf("%.3f", rtt),
fmt.Sprintf("%.5f", loss),
fmt.Sprintf("%.3f", ctx.DeviceCpu),
fmt.Sprintf("%.3f", ecpu),
fmt.Sprintf("%.1f", size),
fmt.Sprintf("%.1f", slo),
a,
fmt.Sprintf("%.3f", p.lat),
fmt.Sprintf("%.5f", p.energy),
fmt.Sprintf("%.3f", p.p95),
fmt.Sprintf("%d", viol),
fmt.Sprintf("%d", time.Now().Unix()),
})
total++
}
}

// policy 4: AlwaysEdgeMed
{
a := "edge1:med"
if p, ok := cache[a]; ok {
viol := 0; if p.p95 > float64(ctx.SloP95Ms) { viol = 1 }
w.Write([]string{
"AlwaysEdgeMed", ctx.TenantId, ctx.AppId,
fmt.Sprintf("%.3f", bw),
fmt.Sprintf("%.3f", rtt),
fmt.Sprintf("%.5f", loss),
fmt.Sprintf("%.3f", ctx.DeviceCpu),
fmt.Sprintf("%.3f", ecpu),
fmt.Sprintf("%.1f", size),
fmt.Sprintf("%.1f", slo),
a,
fmt.Sprintf("%.3f", p.lat),
fmt.Sprintf("%.5f", p.energy),
fmt.Sprintf("%.3f", p.p95),
fmt.Sprintf("%d", viol),
fmt.Sprintf("%d", time.Now().Unix()),
})
total++
}
}

// policy 5: AlwaysCloudLow
{
a := "cloud1:low"
if p, ok := cache[a]; ok {
viol := 0; if p.p95 > float64(ctx.SloP95Ms) { viol = 1 }
w.Write([]string{
"AlwaysCloudLow", ctx.TenantId, ctx.AppId,
fmt.Sprintf("%.3f", bw),
fmt.Sprintf("%.3f", rtt),
fmt.Sprintf("%.5f", loss),
fmt.Sprintf("%.3f", ctx.DeviceCpu),
fmt.Sprintf("%.3f", ecpu),
fmt.Sprintf("%.1f", size),
fmt.Sprintf("%.1f", slo),
a,
fmt.Sprintf("%.3f", p.lat),
fmt.Sprintf("%.5f", p.energy),
fmt.Sprintf("%.3f", p.p95),
fmt.Sprintf("%d", viol),
fmt.Sprintf("%d", time.Now().Unix()),
})
total++
}
}
}
}
}
}
}
}

w.Flush()
if err := w.Error(); err != nil { log.Fatal(err) }
log.Printf("wrote %d policy-records to experiments/results_policies.csv", total)
}
