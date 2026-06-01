package main

import (
	"testing"
	"time"
)

func TestEncodeDeterministic(t *testing.T) {
	result1 := encode(123456789)
	result2 := encode(123456789)
	if result1 != result2 {
		t.Errorf("encode is not deterministic: got %s and %s", result1, result2)
	}
}

func TestGenerateCodeLength(t *testing.T) {
	code := GenerateCode()
	if len(code) != 7 {
		t.Errorf("expected code length 7, got %d", len(code))
	}
}

func TestGenerateCodeUnique(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		code := GenerateCode()
		if codes[code] {
			t.Errorf("collision detected: %s generated twice", code)
		}
		codes[code] = true
	}
}

func TestStoreRoundTrip(t *testing.T) {
	store := NewStore()
	store.Save("abc1234", "https://google.com")
	url, found := store.Get("abc1234")
	if !found {
		t.Error("expected to find url, got not found")
	}
	if url != "https://google.com" {
		t.Errorf("expected https://google.com, got %s", url)
	}
}

func TestStoreMissingCode(t *testing.T) {
	store := NewStore()
	_, found := store.Get("missing")
	if found {
		t.Error("expected not found for missing code")
	}
}

func TestCacheHitAndMiss(t *testing.T) {
	cache := NewCache()
	cache.Set("abc1234", "https://google.com", 5*time.Minute)

	url, found := cache.Get("abc1234")
	if !found || url != "https://google.com" {
		t.Error("expected cache hit")
	}

	_, found = cache.Get("missing")
	if found {
		t.Error("expected cache miss for missing code")
	}
}

func TestCacheExpiry(t *testing.T) {
	cache := NewCache()
	cache.Set("abc1234", "https://google.com", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, found := cache.Get("abc1234")
	if found {
		t.Error("expected cache miss after expiry")
	}
}

func TestRateLimiterAllows(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 10; i++ {
		if !rl.IsAllowed("user1") {
			t.Errorf("expected allowed on request %d", i+1)
		}
	}
}

func TestRateLimiterBlocks(t *testing.T) {
	rl := NewRateLimiter()
	for i := 0; i < 10; i++ {
		rl.IsAllowed("user1")
	}
	if rl.IsAllowed("user1") {
		t.Error("expected rate limit to block after 10 requests")
	}
}
