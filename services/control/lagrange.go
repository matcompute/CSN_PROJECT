package main

import (
"net/http"
"strconv"
"sync"

"github.com/prometheus/client_golang/prometheus"
)

var (
deciderInstance *deciderServer

mMuSLO     = prometheus.NewGauge(prometheus.GaugeOpts{
Name: "csn_mu_slo", Help: "Current SLO Lagrange multiplier",
})
mGammaFair = prometheus.NewGauge(prometheus.GaugeOpts{
Name: "csn_gamma_fair_ms", Help: "Current fairness penalty (ms per excess resource unit)",
})

handlersOnce sync.Once
)

// registerLagrangeHandlers can be called multiple times; routes will be registered once.
func registerLagrangeHandlers(ds *deciderServer) {
deciderInstance = ds
handlersOnce.Do(func() {
// Read current values
http.HandleFunc("/lagrange/get", func(w http.ResponseWriter, r *http.Request) {
ds.mu.Lock()
muSLO := ds.muSLO
gamma := ds.fairGammaMs
ds.mu.Unlock()
_, _ = w.Write([]byte(
`{"mu_slo":` + strconv.FormatFloat(muSLO, 'f', 6, 64) +
`,"gamma_fair_ms":` + strconv.FormatFloat(gamma, 'f', 6, 64) + `}`))
})

// Update values via query/form (e.g., /lagrange/set?mu_slo=3&gamma_fair_ms=15)
http.HandleFunc("/lagrange/set", func(w http.ResponseWriter, r *http.Request) {
if err := r.ParseForm(); err != nil {
http.Error(w, err.Error(), 400)
return
}
if v := r.Form.Get("mu_slo"); v != "" {
if f, err := strconv.ParseFloat(v, 64); err == nil {
ds.mu.Lock(); ds.muSLO = f; ds.mu.Unlock()
}
}
if v := r.Form.Get("gamma_fair_ms"); v != "" {
if f, err := strconv.ParseFloat(v, 64); err == nil {
ds.mu.Lock(); ds.fairGammaMs = f; ds.mu.Unlock()
}
}
// Update gauges
ds.mu.Lock()
mMuSLO.Set(ds.muSLO)
mGammaFair.Set(ds.fairGammaMs)
ds.mu.Unlock()

w.WriteHeader(http.StatusNoContent)
})
})
}

func init() {
// Register gauges once with Prometheus
prometheus.MustRegister(mMuSLO, mGammaFair)
}
