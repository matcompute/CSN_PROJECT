package main

import (
"context"
"fmt"
"log"
"net"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

type deciderServer struct {
pb.UnimplementedDeciderServer
predictor pb.PredictorClient
}

func (s *deciderServer) Decide(ctx context.Context, req *pb.DecideRequest) (*pb.DecideReply, error) {
bestAction := ""
bestLatency := 1e18

// simple deadline so we don't hang
cctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
defer cancel()

for _, a := range req.FeasibleActions {
// ask predictor for this action
resp, err := s.predictor.Predict(cctx, &pb.PredictRequest{
Ctx:    req.Ctx,
Action: a,
})
if err != nil {
// skip on error
continue
}
if resp.MuLatencyMs < bestLatency {
bestLatency = resp.MuLatencyMs
bestAction = a
}
}

// fallback if none succeeded
if bestAction == "" && len(req.FeasibleActions) > 0 {
bestAction = req.FeasibleActions[0]
}

return &pb.DecideReply{ChosenAction: bestAction, Explore: false}, nil
}

func main() {
// connect to predictor
conn, err := grpc.Dial("127.0.0.1:7001", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil {
log.Fatalf("connect predictor: %v", err)
}
defer conn.Close()

pred := pb.NewPredictorClient(conn)

// start decider server
lis, err := net.Listen("tcp", ":7002")
if err != nil {
log.Fatalf("listen: %v", err)
}
s := grpc.NewServer()
pb.RegisterDeciderServer(s, &deciderServer{predictor: pred})

fmt.Println("Decider listening on :7002 (calling predictor at :7001)")
if err := s.Serve(lis); err != nil {
log.Fatalf("serve: %v", err)
}
}
