package main

import (
"context"
"crypto/rand"
"encoding/binary"
"fmt"
"log"
"math"
mrand "math/rand"
"net"
"strings"
"time"

"google.golang.org/grpc"
pb "github.com/mulat/csn/proto"
)

type deciderServer struct {
pb.UnimplementedDeciderServer
predictor     pb.PredictorClient
lambdaEnergy  float64
alphaSLO      float64
exploreStdCap float64
epsilon       float64 // epsilon-greedy explore prob
}

func parseKindTier(a string) (kind, tier string) {
kind, tier = "edge", "med"
if a == "" { return }
parts := strings.Split(a, ":")
k := parts[0]
t := "med"
if len(parts) > 1 { t = parts[1] }
if strings.HasPrefix(k, "edge") { k = "edge" }
if strings.HasPrefix(k, "cloud") { k = "cloud" }
if k != "local" && k != "edge" && k != "cloud" { k = "edge" }
if t != "low" && t != "med" && t != "high" { t = "med" }
return k, t
}

func actionCostMs(a string) float64 {
kind, tier := parseKindTier(a)
kindCost := map[string]float64{
"local": 0,
"edge":  15,
"cloud": 40,
}[kind]
tierCost := map[string]float64{
"low":  0,
"med":  40,
"high": 120,
}[tier]
return kindCost + tierCost
}

type scored struct {
action string
u     float64
}

func (s *deciderServer) Decide(ctx context.Context, req *pb.DecideRequest) (*pb.DecideReply, error) {
bestAction := ""
bestU := math.Inf(-1)
scores := make([]scored, 0, len(req.FeasibleActions))

cctx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
defer cancel()

jitter := func() float64 { return mrand.NormFloat64() * 0.5 }

for _, a := range req.FeasibleActions {
resp, err := s.predictor.Predict(cctx, &pb.PredictRequest{Ctx: req.Ctx, Action: a})
if err != nil { continue }

mLat := float64(resp.MuLatencyMs)
vLat := math.Max(1e-6, float64(resp.VarLatency))
mEn  := float64(resp.MuEnergyJ)
vEn  := math.Max(1e-6, float64(resp.VarEnergy))
p95  := float64(resp.P95ConformalMs)
slo  := float64(req.Ctx.SloP95Ms)

stdL := math.Min(math.Sqrt(vLat), s.exploreStdCap)
stdE := math.Min(math.Sqrt(vEn),  s.exploreStdCap*0.1)
latSample := mLat + mrand.NormFloat64()*stdL
enSample  := mEn  + mrand.NormFloat64()*stdE

sloPenalty := math.Max(0, p95 - slo)
costMs := actionCostMs(a)

U := -(latSample + s.lambdaEnergy*enSample + s.alphaSLO*sloPenalty + costMs) + jitter()
scores = append(scores, scored{action: a, u: U})

if U > bestU {
bestU = U
bestAction = a
}
}

// epsilon-greedy: with prob epsilon, pick a random *different* feasible action (if any)
if len(scores) > 1 && mrand.Float64() < s.epsilon {
// find a non-best index
idx := mrand.Intn(len(scores))
for scores[idx].action == bestAction && len(scores) > 1 {
idx = mrand.Intn(len(scores))
}
bestAction = scores[idx].action
}

if bestAction == "" && len(req.FeasibleActions) > 0 {
bestAction = req.FeasibleActions[0]
}

// Explore flag reflects epsilon usage (coarse)
explore := false
if mrand.Float64() < s.epsilon { explore = true }

return &pb.DecideReply{ChosenAction: bestAction, Explore: explore}, nil
}

func main() {
// seed RNG
var b [8]byte
if _, err := rand.Read(b[:]); err == nil {
mrand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
} else {
mrand.Seed(time.Now().UnixNano())
}

conn, err := grpc.Dial("127.0.0.1:7001", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil { log.Fatalf("connect predictor: %v", err) }
defer conn.Close()
pred := pb.NewPredictorClient(conn)

lis, err := net.Listen("tcp", ":7002")
if err != nil { log.Fatalf("listen: %v", err) }
s := grpc.NewServer()
pb.RegisterDeciderServer(s, &deciderServer{
predictor:     pred,
lambdaEnergy:  80.0,
alphaSLO:      4.0,
exploreStdCap: 8.0,
epsilon:       0.10, // 10% exploration
})

fmt.Println("Decider listening on :7002 with TS + epsilon-greedy")
if err := s.Serve(lis); err != nil { log.Fatalf("serve: %v", err) }
}
