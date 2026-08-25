// @note #arch-20260821-014 issue status=open priority=P2 tags=#arch,#concurrency : Lock ordering inconsistency in ratestore
//
// The evictStale method (line 150) acquires shard.mu and then bucket.mu,
// while other methods (CheckAndConsume, Increment) acquire bucket.mu first.
//
// This lock ordering inconsistency could lead to deadlocks if concurrent
// operations are happening. For example:
// - Thread A: evictStale holds shard.mu, waits for bucket.mu
// - Thread B: CheckAndConsume holds bucket.mu, waits for shard.mu
//
// Resolution: Use a consistent lock ordering throughout, or use a different
// eviction strategy that doesn't require holding both locks simultaneously.
// @note #bench-20260821-001 todo status=open priority=P1 tags=#benchmark,#performance : Rate limiting needs benchmarks
// @assignee opencode
//
// Rate limiting is critical for HFT fair-use policies and IoT device
// management. Current implementation lacks benchmarks for:
//
// 1. CheckAndConsume under high concurrency (1000+ goroutines)
// 2. Memory usage with millions of keys
// 3. Eviction performance under load
// 4. Comparison with Redis-backed rate limiting
//
// For HFT: Rate limiting latency directly impacts trading performance.
// For IoT: Rate limiting must handle thousands of devices simultaneously.
//
// Resolution: Add benchmarks in ratestore/inmemory_bench_test.go:
// - BenchmarkCheckAndConsume_Serial
// - BenchmarkCheckAndConsume_Parallel
// - BenchmarkMemoryUsage_1MKeys
// - BenchmarkEviction_UnderLoad
package ratestore

import (
	"context"
	"math"
	"sync"
	"time"
)

const numShards = 64

var Now = time.Now

// SetNow overrides the clock for testing. Returns a restore function.
func SetNow(fn func() time.Time) func() {
	orig := Now
	Now = fn
	return func() { Now = orig }
}

type bucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
}

type shard struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
}

type InMemoryStore struct {
	shards  [numShards]*shard
	closeCh chan struct{}
}

func New() *InMemoryStore {
	s := &InMemoryStore{closeCh: make(chan struct{})}
	for i := range numShards {
		s.shards[i] = &shard{buckets: make(map[string]*bucket)}
	}
	go s.evictLoop()
	return s
}

func (s *InMemoryStore) Close() {
	close(s.closeCh)
}

func (s *InMemoryStore) shardFor(key string) *shard {
	h := hash(key)
	return s.shards[h%numShards]
}

func (s *InMemoryStore) CheckAndConsume(_ context.Context, key string, burst int64, tokensPerPeriod int64, period time.Duration) (int, bool, error) {
	if burst <= 0 {
		return 0, false, nil
	}
	sh := s.shardFor(key)
	b := s.getOrCreate(sh, key, burst)
	b.mu.Lock()
	defer b.mu.Unlock()

	s.refill(b, float64(burst), float64(tokensPerPeriod), period)

	if b.tokens < 1 {
		return int(math.Floor(b.tokens)), false, nil
	}

	b.tokens--
	return int(math.Floor(b.tokens)), true, nil
}

func (s *InMemoryStore) Increment(_ context.Context, key string, window time.Duration) (int, error) {
	sh := s.shardFor(key)
	b := s.getOrCreate(sh, key, 0)
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tokens < 1 || Now().Sub(b.lastRefill) >= window {
		b.tokens = 0
		b.lastRefill = Now()
	}

	b.tokens++
	return int(b.tokens), nil
}

func (s *InMemoryStore) Reset(_ context.Context, key string) error {
	sh := s.shardFor(key)
	sh.mu.Lock()
	delete(sh.buckets, key)
	sh.mu.Unlock()
	return nil
}

func (s *InMemoryStore) getOrCreate(sh *shard, key string, burst int64) *bucket {
	sh.mu.RLock()
	b, ok := sh.buckets[key]
	sh.mu.RUnlock()
	if ok {
		return b
	}
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if b, ok = sh.buckets[key]; ok {
		return b
	}
	b = &bucket{tokens: float64(burst), lastRefill: Now()}
	sh.buckets[key] = b
	return b
}

func (s *InMemoryStore) refill(b *bucket, capacity, tokensPerPeriod float64, period time.Duration) {
	elapsed := Now().Sub(b.lastRefill)
	if elapsed < 0 {
		b.lastRefill = Now()
		return
	}
	rate := tokensPerPeriod / period.Seconds()
	add := rate * elapsed.Seconds()
	if add <= 0 {
		return
	}
	b.tokens = math.Min(capacity, b.tokens+add)
	b.lastRefill = Now()
}

func (s *InMemoryStore) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.evictStale()
		case <-s.closeCh:
			return
		}
	}
}

// @note #review-20260821-018 issue status=open priority=P2 tags=#review,#concurrency : Potential deadlock in evictStale
// The evictStale method acquires shard.mu and then bucket.mu, while other
// methods acquire bucket.mu first (e.g., CheckAndConsume, Increment). This
// lock ordering inconsistency could lead to deadlocks if concurrent operations
// are happening.
//
// Consider using a consistent lock ordering throughout, or using a different
// eviction strategy that doesn't require holding both locks simultaneously.
func (s *InMemoryStore) evictStale() {
	deadline := Now().Add(-10 * time.Minute)
	for _, sh := range s.shards {
		// Collect stale keys under shard lock, then delete after releasing it.
		// This avoids acquiring bucket.mu while holding shard.mu, which would
		// invert the lock order used by CheckAndConsume/Increment (bucket.mu
		// first, then shard.mu.RLock in getOrCreate).
		sh.mu.RLock()
		var stale []string
		for k, b := range sh.buckets {
			b.mu.Lock()
			if b.lastRefill.Before(deadline) {
				stale = append(stale, k)
			}
			b.mu.Unlock()
		}
		sh.mu.RUnlock()

		if len(stale) > 0 {
			sh.mu.Lock()
			for _, k := range stale {
				if b, ok := sh.buckets[k]; ok {
					b.mu.Lock()
					if b.lastRefill.Before(deadline) {
						delete(sh.buckets, k)
					}
					b.mu.Unlock()
				}
			}
			sh.mu.Unlock()
		}
	}
}

func hash(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}
