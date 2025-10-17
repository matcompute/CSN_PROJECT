package main

import (
"context"
"fmt"
"log"
"net"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

type predictorServer struct {
pb.UnimplementedPredictorServer
}

func (s *predictorServer) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictReply, error) {
// Dummy predictions so the pipeline runs
return &pb.PredictReply{
MuLatencyMs:    100,
VarLatency:     400,
MuEnergyJ:      0.2,
VarEnergy:      0.01,
P95ConformalMs: 120,
}, nil
}

func main() {
lis, err := net.Listen("tcp", ":7001")
if err != nil {
log.Fatalf("listen: %v", err)
}
grpcServer := grpc.NewServer()
pb.RegisterPredictorServer(grpcServer, &predictorServer{})
fmt.Println("Predictor listening on :7001")
if err := grpcServer.Serve(lis); err != nil {
log.Fatalf("serve: %v", err)
}
}
