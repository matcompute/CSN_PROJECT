package main

import (
"context"
"encoding/csv"
"fmt"
"log"
"math"
"os"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

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

// open CSV
if err := os.MkdirAll("experiments", 0o755); err != nil { log.Fatal(err) }
f, err := os.Create("experiments/results.csv")
if err != nil { log.Fatal(err) }
defer f.Close()
w := csv.NewWriter(f)
defer w.Flush()

// header
w.Write([]string{
"tenant","app","bw_mbps","rtt_ms","loss","device_cpu","edge_cpu","input_kb","slo_p95_ms",
"chosen_action","mu_latency_ms","mu_energy_j","p95_conformal_ms","slo_viol","ts",
})

feasible := []string{"local:med","edge1:low","edge1:med","edge1:high","cloud1:low"}

// simple grid
bwList   := []float64{5, 20, 50, 100}
rttList  := []float64{10, 40, 80, 120}
lossList := []float64{0.0, 0.003, 0.01}
ecpuList := []float64{0.2, 0.6, 0.9}
sizeList := []float64{64, 256, 1024, 1536}
sloList  := []float64{100, 140, 180, 220}

total := 0
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
cctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
resp, err := dec.Decide(cctx, &pb.DecideRequest{Ctx: ctx, FeasibleActions: feasible})
cancel()
if err != nil {
log.Printf("decide err: %v", err)
continue
}

// ask predictor for chosen action to log metrics
pctx, pcancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
pred, err := predictor.Predict(pctx, &pb.PredictRequest{Ctx: ctx, Action: resp.ChosenAction})
pcancel()
if err != nil {
log.Printf("predict err: %v", err)
continue
}

viol := 0
if pred.P95ConformalMs > ctx.SloP95Ms + 1e-9 {
viol = 1
}

rec := []string{
ctx.TenantId, ctx.AppId,
fmt.Sprintf("%.3f", bw),
fmt.Sprintf("%.3f", rtt),
fmt.Sprintf("%.5f", loss),
fmt.Sprintf("%.3f", ctx.DeviceCpu),
fmt.Sprintf("%.3f", ecpu),
fmt.Sprintf("%.1f", size),
fmt.Sprintf("%.1f", slo),
resp.ChosenAction,
fmt.Sprintf("%.3f", pred.MuLatencyMs),
fmt.Sprintf("%.5f", pred.MuEnergyJ),
fmt.Sprintf("%.3f", pred.P95ConformalMs),
fmt.Sprintf("%d", viol),
fmt.Sprintf("%d", time.Now().Unix()),
}
if err := w.Write(rec); err != nil {
log.Printf("csv write err: %v", err)
}
total++
}
}
}
}
}
}
w.Flush()
if err := w.Error(); err != nil { log.Fatal(err) }
log.Printf("wrote %d records to experiments/results.csv", total)
// quick sanity: count violation rate
// (we'll do detailed analysis/plots in the next step)
_ = math.NaN() // placeholder so we imported math
}
