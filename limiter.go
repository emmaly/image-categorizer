package main

import (
	"time"
)

type Limiter struct {
	rate   int
	tokens chan struct{}
	ticker *time.Ticker
}

func NewLimiter(ratePerMinute int) *Limiter {
	l := &Limiter{
		rate:   ratePerMinute,
		tokens: make(chan struct{}, ratePerMinute),
		ticker: time.NewTicker(time.Minute / time.Duration(ratePerMinute)),
	}

	// Fill the bucket
	for i := 0; i < ratePerMinute; i++ {
		l.tokens <- struct{}{}
	}

	// Start the token replenishment goroutine
	go l.refillTokens()

	return l
}

func (l *Limiter) refillTokens() {
	for range l.ticker.C {
		select {
		case l.tokens <- struct{}{}:
			// Token added
		default:
			// Bucket is full, do nothing
		}
	}
}

func (l *Limiter) Wait() {
	<-l.tokens
}
