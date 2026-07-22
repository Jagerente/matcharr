package session

import (
	"context"
	"log/slog"
	"time"

	"github.com/Jagerente/matcharr/internal/service/deck"
)

const (
	// janitorInterval has to be finer than the shortest deadline the sweep
	// enforces, or a grace period silently becomes a grace period plus a tick.
	janitorInterval = time.Minute
	// asyncFetchTimeout bounds a deck extension that runs with no client
	// request attached, after a kick or leave resolved the vote.
	asyncFetchTimeout = 30 * time.Second
)

type DeckBuilder interface {
	HasPreset(id string) bool
	Build(ctx context.Context, req deck.Request) (deck.Result, error)
}

type Service struct {
	store       *Store
	decks       DeckBuilder
	limits      Limits
	logger      *slog.Logger
	subscribers []Subscriber
	clock       func() time.Time
}

type Option func(*Service)

func WithClock(clock func() time.Time) Option {
	return func(s *Service) { s.clock = clock }
}

func New(decks DeckBuilder, limits Limits, logger *slog.Logger, opts ...Option) *Service {
	s := &Service{
		decks:  decks,
		limits: limits,
		logger: logger,
		clock:  time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.store = newStore(limits, logger, func() time.Time { return s.clock() }, s.dispatch)
	return s
}

// Subscribe runs at assembly time only, before any traffic, so the slice
// needs no lock.
func (s *Service) Subscribe(fn Subscriber) {
	s.subscribers = append(s.subscribers, fn)
}

func (s *Service) dispatch(code string, events []Event) {
	for _, sub := range s.subscribers {
		for _, ev := range events {
			sub(code, ev)
		}
	}
}

func (s *Service) Stats() Stats { return s.store.Stats() }

func (s *Service) RunJanitor(ctx context.Context) {
	ticker := time.NewTicker(janitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.store.Cleanup()
		}
	}
}
