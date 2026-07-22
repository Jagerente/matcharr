package session

import "time"

// Limits bound what a single instance keeps alive. MaxSessions is the guard
// for a public deployment, where nothing stops a stranger from opening
// lobbies in a loop; zero means unlimited. TTL, EmptyGrace and FinishedTTL
// are the three ways a session can die, see Store.expiryReason.
type Limits struct {
	TTL             time.Duration
	EmptyGrace      time.Duration
	FinishedTTL     time.Duration
	MaxSessions     int
	MaxParticipants int
	MaxDeck         int
}

const (
	minDeckLimit     = 10
	defaultDeckLimit = 50
)

// deckRange collapses when the ceiling sits below the usual floor: that
// is a deliberately tiny instance, not a contradiction, and rejecting
// every session would be worse.
func (l Limits) deckRange() (low, high int) {
	return min(minDeckLimit, l.MaxDeck), l.MaxDeck
}

func (l Limits) defaultDeck() int { return min(defaultDeckLimit, l.MaxDeck) }
