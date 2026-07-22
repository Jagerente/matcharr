package domain

import "time"

type Match struct {
	Movie     Movie
	MatchedAt time.Time
	// Votes is a snapshot of who liked and who passed at match time,
	// matched movies are past the point of vote secrecy.
	Votes map[string]Verdict
}

type NearMatch struct {
	Movie  Movie
	Likes  int
	Needed int
	Votes  map[string]Verdict
}

type ParticipantStats struct {
	ID    string
	Name  string
	Likes int
	Seen  int
}

type Results struct {
	Matches     []Match
	NearMatches []NearMatch
	Stats       []ParticipantStats
}
