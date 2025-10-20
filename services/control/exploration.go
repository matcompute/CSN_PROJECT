package main

import (
"math"
"time"

"github.com/prometheus/client_golang/prometheus"
)

var (
mExploreEpsilon = prometheus.NewGauge(prometheus.GaugeOpts{
Name: "csn_explore_epsilon",
Help: "Current exploration rate epsilon",
})
mViolRate = prometheus.NewGauge(prometheus.GaugeOpts{
Name: "csn_viol_rate",
Help: "Rolling SLO violation rate",
})
)

func init() {
prometheus.MustRegister(mExploreEpsilon, mViolRate)
}

func (s *deciderServer) currentViolRate() float64 {
s.mu.Lock()
defer s.mu.Unlock()
return s.currentViolRateLocked()
}

func (s *deciderServer) startExplorationGovernor() {
// seed gauges
mExploreEpsilon.Set(s.epsilon)
mViolRate.Set(s.currentViolRate())

go func() {
t := time.NewTicker(5 * time.Second)
defer t.Stop()
for range t.C {
rate := s.currentViolRate()
// shrink epsilon quickly on violations; grow slowly when healthy
if rate > s.targetEps {
s.epsilon = math.Max(0.01, s.epsilon*0.5)
} else {
s.epsilon = math.Min(0.20, s.epsilon*1.05)
}
mExploreEpsilon.Set(s.epsilon)
mViolRate.Set(rate)
}
}()
}
