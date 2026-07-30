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

func (s *InMemoryStore) evictStale() {
	deadline := Now().Add(-10 * time.Minute)
	for _, sh := range s.shards {
		sh.mu.Lock()
		for k, b := range sh.buckets {
			b.mu.Lock()
			if b.lastRefill.Before(deadline) {
				delete(sh.buckets, k)
			}
			b.mu.Unlock()
		}
		sh.mu.Unlock()
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
