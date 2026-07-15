package greetings

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
)

type Salutation struct {
	ID      string `json:"id"`
	Phrase  string `json:"phrase"`
	Creator string `json:"creator"`
}

type GreetingStore struct {
	mu       sync.RWMutex
	salutes  map[string]Salutation
	phrases  []string
	nextID   int
}

func NewGreetingStore() *GreetingStore {
	return &GreetingStore{
		salutes: make(map[string]Salutation),
		phrases: []string{
			"Hello", "Hi", "Hey", "Greetings", "Howdy",
			"Salutations", "Yo", "What's up", "Nice to meet",
		},
		nextID: 1,
	}
}

func (s *GreetingStore) Create(_ context.Context, phrase, creator string) (Salutation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("sal_%d", s.nextID)
	s.nextID++
	g := Salutation{ID: id, Phrase: phrase, Creator: creator}
	s.salutes[id] = g
	s.phrases = append(s.phrases, phrase)
	return g, nil
}

func (s *GreetingStore) Get(_ context.Context, id string) (Salutation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.salutes[id]
	return g, ok
}

func (s *GreetingStore) List(_ context.Context) []Salutation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Salutation, 0, len(s.salutes))
	for _, g := range s.salutes {
		out = append(out, g)
	}
	return out
}

func (s *GreetingStore) RandomPhrase() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phrases[rand.Intn(len(s.phrases))]
}

func (s *GreetingStore) Generate(ctx context.Context, name, salutationID string) string {
	var phrase string
	if salutationID != "" {
		g, ok := s.Get(ctx, salutationID)
		if !ok {
			return fmt.Sprintf("Welcome, %s!", name)
		}
		phrase = g.Phrase
	} else {
		phrase = s.RandomPhrase()
	}
	return fmt.Sprintf("%s, %s!", phrase, name)
}
