package main

import (
"context"
"fmt"
"log"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

func main() {
conn, err := grpc.Dial("127.0.0.1:7002", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil { log.Fatalf("connect decider: %v", err) }
defer conn.Close()
dec := pb.NewDeciderClient(conn)

ctxFixed := &pb.Context{
TenantId:   "tenantA",
AppId:      "app1",
BwMbps:     20,   // fixed context
        RttMs:      50,
        Loss:       0.002,
        DeviceCpu:  0.5,
        BatterySoc: 0.7,
        EdgeCpu:    0.6,
        InputKb:    512,
        SloP95Ms:   140,
}
feasible := []string{"local:med", "edge1:low", "edge1:med", "edge1:high", "cloud1:low"}

counts := map[string]int{}
N := 50
for i := 0; i < N; i++ {
cctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
resp, err := dec.Decide(cctx, &pb.DecideRequest{Ctx: ctxFixed, FeasibleActions: feasible})
cancel()
if err != nil { log.Printf("decide err: %v", err); continue }
counts[resp.ChosenAction]++
}

fmt.Println("=== Thompson Sampling check on fixed context ===")
for _, a := range feasible {
fmt.Printf("%-12s : %d\n", a, counts[a])
}
}
