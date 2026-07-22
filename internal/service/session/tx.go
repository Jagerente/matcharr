package session

import (
	"time"

	"github.com/Jagerente/matcharr/internal/domain"
)

// Tx is one transaction over a session, always executed under the store
// lock. Events recorded here are dispatched after the lock is released,
// and only when the transaction succeeds.
type Tx struct {
	store   *Store
	code    string
	session *domain.Session
	now     time.Time
	actor   *domain.Participant
	events  []Event
}

func (tx *Tx) Session() *domain.Session { return tx.session }

func (tx *Tx) Now() time.Time { return tx.now }

// Actor is the authenticated participant, nil inside internal follow-ups
// that run without a token.
func (tx *Tx) Actor() *domain.Participant { return tx.actor }

func (tx *Tx) Authenticate(token string) (*domain.Participant, error) {
	p := tx.session.ParticipantByToken(token)
	if p == nil {
		return nil, errTokenInvalid
	}
	tx.session.Touch(tx.now)
	tx.actor = p
	return p, nil
}

func (tx *Tx) Emit(eventType EventType, data any) {
	tx.events = append(tx.events, Event{Type: eventType, State: buildView(tx.session), Data: data})
}

// Fetching reports whether a deck build for this session is in flight.
// The flag lives in the store, not the domain: it is concurrency
// bookkeeping, not session state.
func (tx *Tx) Fetching() bool { return tx.store.fetching[tx.code] }

func (tx *Tx) MarkFetching(inFlight bool) {
	if inFlight {
		tx.store.fetching[tx.code] = true
		return
	}
	delete(tx.store.fetching, tx.code)
}
