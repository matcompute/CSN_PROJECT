package main

import (
"context"
"fmt"
"log"
"math/rand"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

func main() {
rand.Seed(time.Now().UnixNano())

conn, err := grpc.Dial("127.0.0.1:7002", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil { log.Fatalf("connect decider: %v", err) }
defer conn.Close()
dec := pb.NewDeciderClient(conn)

feasible := []string{"local:med", "edge1:low", "edge1:med", "edge1:high", "cloud1:low"}

counts := map[string]int{}
N := 50
for i := 0; i < N; i++ {
ctx := &pb.Context{
TenantId:   "tenantA",
AppId:      "app1",
BwMbps:     5 + rand.Float64()*100,      // 5..105
RttMs:      10 + rand.Float64()*100,     // 10..110
Loss:       rand.Float64()*0.01,         // 0..1%
DeviceCpu:  0.2 + rand.Float64()*0.6,    // 0.2..0.8
BatterySoc: 0.4 + rand.Float64()*0.6,    // 0.4..1.0
EdgeCpu:    0.2 + rand.Float64()*0.6,    // 0.2..0.8
InputKb:    64 + rand.Float64()*1024,    // 64..1088 KB
SloP95Ms:   100 + rand.Float64()*120,    // 100..220 ms
}
cctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
resp, err := dec.Decide(cctx, &pb.DecideRequest{Ctx: ctx, FeasibleActions: feasible})
cancel()
if err != nil { log.Printf("decide err: %v", err); continue }
counts[resp.ChosenAction]++
}

// print histogram
fmt.Println("=== Decision histogram over", N, "runs ===")
for _, a := range feasible {
fmt.Printf("%-12s : %d\n", a, counts[a])
}
}
