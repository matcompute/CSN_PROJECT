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
// connect to decider (assumes predictor is already running)
conn, err := grpc.Dial("127.0.0.1:7002", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil {
log.Fatalf("connect decider: %v", err)
}
defer conn.Close()
dec := pb.NewDeciderClient(conn)

// sample context (we'll wire real features later)
ctx := &pb.Context{
TenantId:   "tenantA",
AppId:      "app1",
BwMbps:     20,
RttMs:      40,
Loss:       0.0,
DeviceCpu:  0.35,
BatterySoc: 0.80,
EdgeCpu:    0.50,
InputKb:    256,
SloP95Ms:   120,
}

feasible := []string{"local:med", "edge1:low", "edge1:med", "cloud1:low"}

// call decide with a short timeout
cctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
defer cancel()
resp, err := dec.Decide(cctx, &pb.DecideRequest{Ctx: ctx, FeasibleActions: feasible})
if err != nil {
log.Fatalf("decide error: %v", err)
}

fmt.Printf("Chosen action: %s (explore=%v)\n", resp.ChosenAction, resp.Explore)
}
