package main

import (
"sync"
"time"
)

type tokenBucket struct {
rate      float64       // tokens per second
burst     float64       // max bucket
tokens    float64
lastFill  time.Time
mu        sync.Mutex
}

func newBucket(rate, burst float64) *tokenBucket {
return &tokenBucket{rate: rate, burst: burst, tokens: burst, lastFill: time.Now()}
}

func (b *tokenBucket) allow(cost float64) bool {
b.mu.Lock(); defer b.mu.Unlock()
now := time.Now()
elapsed := now.Sub(b.lastFill).Seconds()
b.tokens = minF(b.burst, b.tokens + elapsed*b.rate)
b.lastFill = now
if b.tokens >= cost {
b.tokens -= cost
return true
}
return false
}

func minF(a,b float64) float64 { if a<b { return a }; return b }

type quotaManager struct {
mu sync.Mutex
buckets map[string]*tokenBucket
rate float64
burst float64
}

func newQuotaManager(rate, burst float64) *quotaManager {
return &quotaManager{buckets: make(map[string]*tokenBucket), rate: rate, burst: burst}
}

func (q *quotaManager) allow(tenant string, cost float64) bool {
q.mu.Lock(); defer q.mu.Unlock()
b, ok := q.buckets[tenant]
if !ok {
b = newBucket(q.rate, q.burst)
q.buckets[tenant] = b
}
return b.allow(cost)
}
