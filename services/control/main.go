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
"net/http"
"os"
"strconv"
"strings"
"sync"
"time"

"github.com/prometheus/client_golang/prometheus/promhttp"
"google.golang.org/grpc"

pb "github.com/mulat/csn/proto"
)

// global capacity poller (reads operator /metrics)
var capPoller *CapPoller

// --- helpers ----------------------------------------------------------

func parseKindTier(a string) (kind, tier string) {
kind, tier = "edge", "med"
if a == "" {
return
}
parts := strings.Split(a, ":")
k := parts[0]
t := "med"
if len(parts) > 1 {
t = parts[1]
}
if strings.HasPrefix(k, "edge") {
k = "edge"
}
if strings.HasPrefix(k, "cloud") {
k = "cloud"
}
if k != "local" && k != "edge" && k != "cloud" {
k = "edge"
}
if t != "low" && t != "med" && t != "high" {
t = "med"
}
return k, t
}

func actionCostMs(a string) float64 {
kind, tier := parseKindTier(a)
kindCost := map[string]float64{"local": 0, "edge": 15, "cloud": 40}[kind]
tierCost := map[string]float64{"low": 0, "med": 40, "high": 120}[tier]
return kindCost + tierCost
}

func actionCostMsWithCap(a string, base float64, capFactor float64) float64 {
k, _ := parseKindTier(a)
if k == "edge" {
return base * capFactor
}
return base
}

func resourceIntensity(a string) float64 {
kind, tier := parseKindTier(a)
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

// --- Decider server ----------------------------------------------------------

type deciderServer struct {
// resilience
brk *circuitBreaker

// gRPC
pb.UnimplementedDeciderServer
predictor pb.PredictorClient

// objective weights
lambdaEnergy  float64
alphaSLOBase  float64
exploreStdCap float64
epsilon       float64
useConformal  bool

// fairness EWMA
mu          sync.Mutex
tenantEWMA  map[string]float64
ewmaAlpha   float64
fairGammaMs float64

// SLO primal-dual
muSLO       float64
targetEps   float64
winSize     int
violWin     []int
winIdx      int
eta         float64
lastUpdate  time.Time
updateEvery time.Duration

// Admission/Quota
quota *quotaManager
}

func (s *deciderServer) fairnessPenalty(tenant string, a string) float64 {
s.mu.Lock()
defer s.mu.Unlock()
if s.tenantEWMA == nil {
s.tenantEWMA = make(map[string]float64)
}
ri := resourceIntensity(a)
prev := s.tenantEWMA[tenant]
newv := s.ewmaAlpha*ri + (1.0-s.ewmaAlpha)*prev
if prev == 0 {
newv = ri
}
s.tenantEWMA[tenant] = newv
sum := 0.0
for _, v := range s.tenantEWMA {
sum += v
}
mean := sum / float64(len(s.tenantEWMA))
over := newv - mean
if over <= 0 {
return 0
}
return s.fairGammaMs * over
}

func (s *deciderServer) recordViolation(v int) {
s.mu.Lock()
defer s.mu.Unlock()
if len(s.violWin) == 0 {
s.violWin = make([]int, s.winSize)
}
s.violWin[s.winIdx%len(s.violWin)] = v
s.winIdx++
now := time.Now()
if now.Sub(s.lastUpdate) >= s.updateEvery {
rate := s.currentViolRateLocked()
s.muSLO = math.Max(0, s.muSLO+s.eta*(rate-s.targetEps))
s.lastUpdate = now
}
}

func (s *deciderServer) currentViolRateLocked() float64 {
if len(s.violWin) == 0 {
return 0
}
sum := 0
for _, x := range s.violWin {
sum += x
}
return float64(sum) / float64(len(s.violWin))
}

type scored struct{ action string; u float64 }

func capacityFactor() float64 {
edgesUp := 1
if v := strings.TrimSpace(os.Getenv("CSN_EDGES_UP")); v != "" {
if n, err := strconv.Atoi(v); err == nil && n >= 1 {
edgesUp = n
}
}
coef := 0.15
if v := strings.TrimSpace(os.Getenv("CSN_EDGE_CAP_COEF")); v != "" {
if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
coef = f
}
}
return math.Max(0.3, 1.0-coef*float64(edgesUp-1))
}

func (s *deciderServer) Decide(ctx context.Context, req *pb.DecideRequest) (*pb.DecideReply, error) {
// --- Admission/Quota: deny early if tenant exceeds credits ---
tenantID := req.Ctx.GetTenantId()
if tenantID == "" {
tenantID = "default"
}
if s.quota != nil && !s.quota.allow(tenantID, 1.0) {
fallback := "local:low"
if len(req.FeasibleActions) > 0 {
fallback = req.FeasibleActions[0]
}
return &pb.DecideReply{ChosenAction: fallback, Explore: false}, nil
}

// circuit breaker flag (value unused; allows breaker to gate predictor below)
_ = s.brk != nil && !s.brk.allow()

bestAction := ""
bestU := math.Inf(-1)

cctx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
defer cancel()

jitter := func() float64 { return mrand.NormFloat64() * 0.5 }

scores := make([]scored, 0, len(req.FeasibleActions))
type obs struct{ a string; p95, slo float64 }
observed := make([]obs, 0, len(req.FeasibleActions))

cf := 1.0
if capPoller != nil {
cf = capPoller.Factor()
}

for _, a := range req.FeasibleActions {
// circuit breaker: prefer cheapest/local when open
if s.brk != nil && !s.brk.allow() {
if bestAction == "" {
bestAction = a
}
continue
}

resp, err := s.predictor.Predict(cctx, &pb.PredictRequest{Ctx: req.Ctx, Action: a})
if err != nil {
if s.brk != nil {
s.brk.onFailure()
}
continue
}
if s.brk != nil {
s.brk.onSuccess()
}

mLat := float64(resp.MuLatencyMs)
vLat := math.Max(1e-9, float64(resp.VarLatency))
mEn := float64(resp.MuEnergyJ)
p95c := float64(resp.P95ConformalMs)
slo := float64(req.Ctx.SloP95Ms)

if k, _ := parseKindTier(a); k == "edge" {
mLat = mLat * cf
}

stdL := math.Min(math.Sqrt(vLat), s.exploreStdCap)
latSample := mLat + mrand.NormFloat64()*stdL
enSample := mEn

var p95eff float64
if s.useConformal {
p95eff = p95c
} else {
p95eff = mLat + 1.645*math.Sqrt(vLat)
}

sloPenalty := math.Max(0, p95eff-slo)
baseCost := actionCostMs(a)
capF := 1.0
if capPoller != nil {
capF = capPoller.Factor()
}
costMs := actionCostMsWithCap(a, baseCost, capF)
alphaEff := s.alphaSLOBase + s.muSLO

U := -(latSample + s.lambdaEnergy*enSample + alphaEff*sloPenalty + costMs) + jitter()

scores = append(scores, scored{action: a, u: U})
observed = append(observed, obs{a: a, p95: p95eff, slo: slo})

if U > bestU {
bestU = U
bestAction = a
}
}

// ε-greedy
if len(scores) > 1 && mrand.Float64() < s.epsilon {
idx := mrand.Intn(len(scores))
for scores[idx].action == bestAction && len(scores) > 1 {
idx = mrand.Intn(len(scores))
}
bestAction = scores[idx].action
}

if bestAction == "" && len(req.FeasibleActions) > 0 {
bestAction = req.FeasibleActions[0]
}

// fairness recheck
fpen := s.fairnessPenalty(tenantID, bestAction)
if fpen > 0 && len(scores) > 1 {
chosenU := math.Inf(-1)
for _, sc := range scores {
if sc.action == bestAction {
chosenU = sc.u - fpen
break
}
}
for _, sc := range scores {
if sc.action == bestAction {
continue
}
if sc.u > chosenU {
bestAction = sc.action
break
}
}
}

// update violation window
for _, o := range observed {
if o.a == bestAction {
v := 0
if o.p95 > o.slo {
v = 1
}
s.recordViolation(v)
break
}
}

// optional sensing hook (no-op if not present)
postSense(req.Ctx, bestAction)

return &pb.DecideReply{ChosenAction: bestAction, Explore: true}, nil
}

// --- main --------------------------------------------------------------------

func main() {
// strong RNG seed
var b [8]byte
if _, err := rand.Read(b[:]); err == nil {
mrand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
} else {
mrand.Seed(time.Now().UnixNano())
}

useConf := true
if v := strings.TrimSpace(os.Getenv("CSN_USE_CONFORMAL")); v != "" {
if v == "0" || strings.ToLower(v) == "false" || v == "off" {
useConf = false
}
}

// predictor proxy
conn, err := grpc.Dial("127.0.0.1:7001", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
if err != nil {
log.Fatalf("connect predictor: %v", err)
}
defer conn.Close()
pred := pb.NewPredictorClient(conn)

// capacity poller from Operator metrics
opURL := strings.TrimSpace(os.Getenv("OP_METRICS_URL"))
if opURL == "" {
opURL = "http://127.0.0.1:9103/metrics"
}
coef := 0.15
if v := strings.TrimSpace(os.Getenv("CSN_EDGE_CAP_COEF")); v != "" {
if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
coef = f
}
}
capPoller = NewCapPoller(opURL, coef, 0.3)
capPoller.Start()

// metrics + control HTTP (lagrange.go registers handlers on default mux)
go func() {
http.Handle("/metrics", promhttp.Handler())
log.Println("Metrics server on :9102")
if err := http.ListenAndServe(":9102", nil); err != nil {
log.Printf("metrics server error: %v", err)
}
}()

// gRPC listener
lis, err := net.Listen("tcp", ":7002")
if err != nil {
log.Fatalf("listen: %v", err)
}

// decider config
s := grpc.NewServer()
ds := &deciderServer{
predictor:     pred,
lambdaEnergy:  80.0,
alphaSLOBase:  2.0,
exploreStdCap: 8.0,
epsilon:       0.10,
useConformal:  useConf,

tenantEWMA:  make(map[string]float64),
ewmaAlpha:   0.3,
fairGammaMs: 10.0,

muSLO:       0.0,
targetEps:   0.10,
winSize:     50,
violWin:     make([]int, 50),
winIdx:      0,
eta:         5.0,
lastUpdate:  time.Now(),
updateEvery: 5 * time.Second,
}

// circuit breaker: 5 consecutive failures -> 10s open
ds.brk = newBreaker(5, 10*time.Second)

// Admission/Quota: 50 rps, burst 100 per-tenant
ds.quota = newQuotaManager(50.0, 100.0)

// exploration governor (exports csn_explore_epsilon)
ds.startExplorationGovernor()

pb.RegisterDeciderServer(s, ds)
fmt.Printf("Decider listening on :7002 (TS+e+fairness+SLO+AQ) useConformal=%v\n", useConf)
if err := s.Serve(lis); err != nil {
log.Fatalf("serve: %v", err)
}
}
