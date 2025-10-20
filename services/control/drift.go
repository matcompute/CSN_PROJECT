package main

import (
"math"
"sync"
"time"

"github.com/prometheus/client_golang/prometheus"
)

var (
mDriftScore = prometheus.NewGauge(prometheus.GaugeOpts{
Name: "csn_drift_score",
Help: "Rolling drift score (|mean - baseline_mean| / baseline_std)",
})
mDriftWindow = prometheus.NewGauge(prometheus.GaugeOpts{
Name: "csn_drift_window",
Help: "Number of samples in the rolling window",
})
)

func init() {
prometheus.MustRegister(mDriftScore, mDriftWindow)
}

type driftWatcher struct {
mu             sync.Mutex
baselineMean   float64
baselineStd    float64
baselineSet    bool

// rolling window stats (Welford)
count   int
mean    float64
M2      float64
winMax  int
}

func newDriftWatcher(win int) *driftWatcher {
return &driftWatcher{winMax: win}
}

func (d *driftWatcher) resetBaseline(mean, std float64) {
d.mu.Lock()
defer d.mu.Unlock()
d.baselineMean = mean
d.baselineStd  = math.Max(std, 1e-6)
d.baselineSet  = true
}

func (d *driftWatcher) addSample(x float64) {
d.mu.Lock()
defer d.mu.Unlock()

// update rolling stats (bounded window via light decay)
d.count++
delta := x - d.mean
d.mean += delta / float64(d.count)
d.M2 += delta * (x - d.mean)

// soft decay to approximate finite window
if d.count > d.winMax {
d.count = int(float64(d.count) * 0.98)
d.M2 *= 0.98
}

var std float64
if d.count > 1 {
std = math.Sqrt(d.M2 / float64(d.count-1))
} else {
std = 1e-6
}

// if no baseline yet, set it from first few seconds
if !d.baselineSet {
// require a minimal warmup
if d.count > 30 {
d.baselineMean = d.mean
d.baselineStd  = math.Max(std, 1e-6)
d.baselineSet  = true
}
mDriftScore.Set(0)
mDriftWindow.Set(float64(d.count))
return
}

// z-score distance from baseline
score := math.Abs(d.mean - d.baselineMean) / math.Max(d.baselineStd, 1e-6)
mDriftScore.Set(score)
mDriftWindow.Set(float64(d.count))
}

func (d *driftWatcher) current() (mean, std float64, n int) {
d.mu.Lock(); defer d.mu.Unlock()
n = d.count
if d.count > 1 {
std = math.Sqrt(d.M2 / float64(d.count-1))
} else {
std = 0
}
mean = d.mean
return
}

// optional periodic logging hook (not required)
func (d *driftWatcher) startLogTicker() {
go func() {
t := time.NewTicker(30 * time.Second)
defer t.Stop()
for range t.C {
d.current()
}
}()
}
