package main

import (
"bufio"
"net/http"
"regexp"
"strconv"
"strings"
"sync"
"time"
)

// CapPoller pulls csn_edges_up from a Prometheus /metrics endpoint
type CapPoller struct {
url       string
coef      float64
floor     float64
client    *http.Client
mu        sync.RWMutex
edgesUp   int
lastFact  float64
reGauge   *regexp.Regexp
}

func NewCapPoller(url string, coef float64, floor float64) *CapPoller {
return &CapPoller{
url:    url,
coef:   coef,
        floor:  floor,
client: &http.Client{Timeout: 1200 * time.Millisecond},
reGauge: regexp.MustCompile(`^csn_edges_up\s+([0-9]+(?:\.[0-9]+)?)$`),
edgesUp: 1,
lastFact: 1.0,
}
}

func (p *CapPoller) Start() {
go func() {
t := time.NewTicker(2 * time.Second)
defer t.Stop()
for range t.C {
p.tick()
}
}()
}

func (p *CapPoller) tick() {
resp, err := p.client.Get(p.url)
if err != nil {
return
}
defer resp.Body.Close()
sc := bufio.NewScanner(resp.Body)
val := 1
for sc.Scan() {
line := strings.TrimSpace(sc.Text())
m := p.reGauge.FindStringSubmatch(line)
if len(m) == 2 {
f, err := strconv.ParseFloat(m[1], 64)
if err == nil && f >= 0 {
val = int(f + 0.0001)
break
}
}
}
// compute factor = max(floor, 1 - coef*(edgesUp-1))
f := 1.0 - p.coef*float64(val-1)
if f < p.floor {
f = p.floor
}
p.mu.Lock()
p.edgesUp = val
p.lastFact = f
p.mu.Unlock()
}

func (p *CapPoller) Factor() float64 {
p.mu.RLock()
defer p.mu.RUnlock()
return p.lastFact
}

func (p *CapPoller) Edges() int {
p.mu.RLock()
defer p.mu.RUnlock()
return p.edgesUp
}
