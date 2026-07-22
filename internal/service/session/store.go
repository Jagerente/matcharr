package session

import (
	"crypto/rand"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

// Store owns the only lock around live sessions, so business code never
// touches a mutex.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*domain.Session
	fetching map[string]bool
	limits   Limits
	logger   *slog.Logger
	clock    func() time.Time
	emit     func(code string, events []Event)
}

func newStore(limits Limits, logger *slog.Logger, clock func() time.Time, emit func(string, []Event)) *Store {
	return &Store{
		sessions: make(map[string]*domain.Session),
		fetching: make(map[string]bool),
		limits:   limits,
		logger:   logger,
		clock:    clock,
		emit:     emit,
	}
}

func (st *Store) Create(build func(code string, now time.Time) *domain.Session) (string, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.limits.MaxSessions > 0 && len(st.sessions) >= st.limits.MaxSessions {
		// The ceiling is worth nothing if dead sessions hold the slots, and
		// waiting for the next sweep would turn a live host away over them.
		st.cleanupLocked()
		if len(st.sessions) >= st.limits.MaxSessions {
			return "", apperr.New(apperr.KindBusy, apperr.ReasonServerBusy,
				"the server is holding its %d session limit", st.limits.MaxSessions)
		}
	}

	code, err := st.newCodeLocked()
	if err != nil {
		return "", err
	}
	st.sessions[code] = build(code, st.clock())
	return code, nil
}

func (st *Store) Update(code string, fn func(tx *Tx) error) error {
	st.mu.Lock()
	s, ok := st.sessions[code]
	if !ok {
		st.mu.Unlock()
		return errNotFound
	}
	tx := &Tx{store: st, code: code, session: s, now: st.clock()}
	err := fn(tx)
	events := tx.events
	st.mu.Unlock()

	if err != nil {
		return err
	}
	st.emit(code, events)
	return nil
}

type Stats struct {
	Sessions     map[domain.Status]int
	Participants int
}

func (st *Store) Stats() Stats {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Every status is seeded so one that drops to zero keeps reporting zero
	// instead of vanishing from the exposition.
	sessions := make(map[domain.Status]int, len(domain.AllStatuses()))
	for _, status := range domain.AllStatuses() {
		sessions[status] = 0
	}

	stats := Stats{Sessions: sessions}
	for _, s := range st.sessions {
		stats.Participants += len(s.Participants)
		sessions[s.Status]++
	}
	return stats
}

func (st *Store) Cleanup() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.cleanupLocked()
}

func (st *Store) cleanupLocked() {
	now := st.clock()
	for code, s := range st.sessions {
		// A deck fetch in flight holds a pointer to this session, so
		// whatever it looks like from here, it is not abandoned yet.
		if st.fetching[code] {
			continue
		}
		reason, expired := st.expiryReason(s, now)
		if !expired {
			continue
		}
		delete(st.sessions, code)
		st.logger.Info("session removed", "code", code, "reason", reason, "status", s.Status)
	}
}

func (st *Store) expiryReason(s *domain.Session, now time.Time) (string, bool) {
	if now.Sub(s.LastSeen) > st.limits.TTL {
		return "ttl", true
	}
	if !s.FinishedAt.IsZero() && now.Sub(s.FinishedAt) > st.limits.FinishedTTL {
		return "results expired", true
	}
	// Sockets alone would condemn a session whose clients fell back to
	// polling, so silence on both channels is what counts as abandoned.
	empty := !s.EmptySince.IsZero() && now.Sub(s.EmptySince) > st.limits.EmptyGrace
	if empty && now.Sub(s.LastSeen) > st.limits.EmptyGrace {
		return "abandoned", true
	}
	return "", false
}

func (st *Store) newCodeLocked() (string, error) {
	for range 100 {
		buf := make([]byte, codeLength)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		code := make([]byte, codeLength)
		for i, b := range buf {
			code[i] = codeAlphabet[int(b)%len(codeAlphabet)]
		}
		if _, taken := st.sessions[string(code)]; !taken {
			return string(code), nil
		}
	}
	return "", errors.New("could not find a free session code")
}
