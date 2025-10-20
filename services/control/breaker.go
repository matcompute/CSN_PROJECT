package main

import (
"sync/atomic"
"time"
)

type circuitBreaker struct {
failBudget int32
maxFails   int32
openUntil  int64 // unix nano until which breaker is open
cooldown   time.Duration
}

func newBreaker(maxFails int, cooldown time.Duration) *circuitBreaker {
return &circuitBreaker{maxFails: int32(maxFails), cooldown: cooldown}
}

func (b *circuitBreaker) allow() bool {
until := time.Unix(0, atomic.LoadInt64(&b.openUntil))
return time.Now().After(until)
}

func (b *circuitBreaker) onSuccess() {
atomic.StoreInt32(&b.failBudget, 0)
}

func (b *circuitBreaker) onFailure() {
f := atomic.AddInt32(&b.failBudget, 1)
if f >= b.maxFails {
atomic.StoreInt32(&b.failBudget, 0)
atomic.StoreInt64(&b.openUntil, time.Now().Add(b.cooldown).UnixNano())
}
}
