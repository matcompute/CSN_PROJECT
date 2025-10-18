package main

import (
"context"
"fmt"
"log"
"math/rand"
"strings"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

func resourceIntensity(a string) float64 {
// must match decider's notion
kind, tier := "edge", "med"
if a != "" {
parts := strings.Split(a, ":")
if len(parts) > 0 {
k := parts[0]
if strings.HasPrefix(k, "edge") { kind = "edge" }
if strings.HasPrefix(k, "cloud") { kind = "cloud" }
if k == "local" { kind = "local" }
}
if len(parts) > 1 {
tier = parts[1]
}
}
base := map[string]float64{"low": 1, "med": 2, "high": 3}[tier]
switch kind {
case "local":
base += 0
case "edge":
base += 1
case "cloud":
base += 2
}
return base
}

func jainsIndex(xs []float64) float64 {
n := float64(len(xs))
if n == 0 { return 0 }
sum := 0.0
sum2 := 0.0
for _, x := range xs {
sum += x
sum2 += x * x
}
if sum2 == 0 { return 0 }
return (sum * sum) / (n * sum2)
}

func main() {
rand.Seed(time.Now().UnixNano())

conn, err := grpc.Dial("127.0.0.1:7002", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil { log.Fatalf("connect decider: %v", err) }
defer conn.Close()
dec := pb.NewDeciderClient(conn)

tenants := []string{"tenantA", "tenantB", "tenantC"}
feasible := []string{"local:med", "edge1:low", "edge1:med", "edge1:high", "cloud1:low"}

// per-tenant stats
type stats struct {
counts map[string]int
totalRI float64
reqs int
}
per := map[string]*stats{}
for _, t := range tenants {
per[t] = &stats{counts: map[string]int{}}
}

// interleaved requests
R := 60 // total per tenant
for i := 0; i < R*len(tenants); i++ {
t := tenants[i%len(tenants)]
ctx := &pb.Context{
TenantId:   t,
AppId:      "app1",
BwMbps:     5 + rand.Float64()*100,
RttMs:      10 + rand.Float64()*100,
Loss:       rand.Float64()*0.01,
DeviceCpu:  0.2 + rand.Float64()*0.6,
BatterySoc: 0.4 + rand.Float64()*0.6,
EdgeCpu:    0.2 + rand.Float64()*0.6,
InputKb:    64 + rand.Float64()*1024,
SloP95Ms:   100 + rand.Float64()*120,
}
cctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
resp, err := dec.Decide(cctx, &pb.DecideRequest{Ctx: ctx, FeasibleActions: feasible})
cancel()
if err != nil { log.Printf("decide err: %v", err); continue }
per[t].counts[resp.ChosenAction]++
per[t].totalRI += resourceIntensity(resp.ChosenAction)
per[t].reqs++
}

// report
fmt.Println("=== Fairness sweep (3 tenants interleaved) ===")
avgRIs := []float64{}
for _, t := range tenants {
st := per[t]
avgRI := st.totalRI / float64(max(1, st.reqs))
avgRIs = append(avgRIs, avgRI)
fmt.Printf("\nTenant %s (reqs=%d, avgRI=%.2f)\n", t, st.reqs, avgRI)
for _, a := range feasible {
fmt.Printf("  %-12s : %d\n", a, st.counts[a])
}
}
fmt.Printf("\nJain's Index over avg resource intensity: %.3f (1.0 = perfectly fair)\n", jainsIndex(avgRIs))
}

func max(a, b int) int {
if a > b { return a }
return b
}
