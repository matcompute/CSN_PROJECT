package main

import (
"bytes"
"context"
"encoding/json"
"fmt"
"log"
"net"
"net/http"
"time"

pb "github.com/mulat/csn/proto"
"google.golang.org/grpc"
)

type httpPredictIn struct {
Features []float64 `json:"features"`
Action   string    `json:"action,omitempty"`
}
type httpPredictOut struct {
MuLatencyMs    float64 `json:"mu_latency_ms"`
VarLatency     float64 `json:"var_latency"`
MuEnergyJ      float64 `json:"mu_energy_j"`
VarEnergy      float64 `json:"var_energy"`
P95ConformalMs float64 `json:"p95_conformal_ms"`
}

type predictorServer struct {
pb.UnimplementedPredictorServer
httpClient *http.Client
baseURL    string
}

func (s *predictorServer) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictReply, error) {
// Map gRPC Context -> feature vector in agreed order
f := []float64{
req.Ctx.BwMbps,
req.Ctx.RttMs,
req.Ctx.Loss,
req.Ctx.DeviceCpu,
req.Ctx.EdgeCpu,
req.Ctx.InputKb,
req.Ctx.SloP95Ms,
}
inp := httpPredictIn{Features: f, Action: req.Action}

body, _ := json.Marshal(inp)
httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/predict", bytes.NewReader(body))
httpReq.Header.Set("Content-Type", "application/json")

resp, err := s.httpClient.Do(httpReq)
if err != nil {
// graceful fallback: return conservative dummy if HTTP fails
log.Printf("http predictor error: %v", err)
return &pb.PredictReply{
MuLatencyMs:    200,
VarLatency:     900,
MuEnergyJ:      0.5,
VarEnergy:      0.05,
P95ConformalMs: 240,
}, nil
}
defer resp.Body.Close()

var out httpPredictOut
if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
log.Printf("decode error: %v", err)
return nil, err
}

return &pb.PredictReply{
MuLatencyMs:    out.MuLatencyMs,
VarLatency:     out.VarLatency,
MuEnergyJ:      out.MuEnergyJ,
VarEnergy:      out.VarEnergy,
P95ConformalMs: out.P95ConformalMs,
}, nil
}

func main() {
cli := &http.Client{Timeout: 500 * time.Millisecond}
s := &predictorServer{
httpClient: cli,
baseURL:    "http://127.0.0.1:8000",
}

lis, err := net.Listen("tcp", ":7001")
if err != nil {
log.Fatalf("listen: %v", err)
}
grpcServer := grpc.NewServer()
pb.RegisterPredictorServer(grpcServer, s)
fmt.Println("Predictor (proxy) listening on :7001, using ONNX service at :8000")
if err := grpcServer.Serve(lis); err != nil {
log.Fatalf("serve: %v", err)
}
}
