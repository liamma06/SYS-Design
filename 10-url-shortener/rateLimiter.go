/*
keep track of requests a user makes to prevent abuse
*/
package main

import (
	"sync"
	"time"
)

type TokenBucket struct {
	capacity   int
	refillRate int
	currTokens int
	lastRefill time.Time
}

type RateLimiter struct {
	buckets map[string]*TokenBucket //map of user/IP to their token bucket
	mu      sync.Mutex
}

// constructor for rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

// before any request make sure to check if user is allowed to make request and take a token
func (rl *RateLimiter) IsAllowed(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	bucket, exists := rl.buckets[userID]

	if !exists {
		bucket = &TokenBucket{
			capacity:   10, //max 10 requests
			refillRate: 1,  //1 token per second
			currTokens: 10, //start with full bucket
			lastRefill: time.Now(),
		}
		rl.buckets[userID] = bucket
	}

	now := time.Now().Unix()
	elapsed := now - bucket.lastRefill.Unix()

	tokensToAdd := int(elapsed) * bucket.refillRate
	if tokensToAdd > 0 {
		bucket.currTokens = min(bucket.capacity, bucket.currTokens+tokensToAdd)
		bucket.lastRefill = time.Now()
	}

	if bucket.currTokens > 0 {
		bucket.currTokens--
		return true
	}
	return false

}
