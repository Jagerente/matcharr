package api

import "time"

type MovieDTO struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	Year          int      `json:"year,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Rating        float64  `json:"rating,omitempty"`
	VoteCount     int      `json:"voteCount,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	PosterPath    string   `json:"posterPath,omitempty"`
	BackdropPath  string   `json:"backdropPath,omitempty"`
	Runtime       int      `json:"runtime,omitempty"`
}

type GenreDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PresetDTO struct {
	ID          string            `json:"id"`
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	MediaType   string            `json:"mediaType"`
}

type FiltersDTO struct {
	GenreIDs  []int64 `json:"genreIds,omitempty"`
	YearFrom  int     `json:"yearFrom,omitempty"`
	YearTo    int     `json:"yearTo,omitempty"`
	RatingMin float64 `json:"ratingMin,omitempty"`
	VotesMin  int     `json:"votesMin,omitempty"`
}

type SettingsDTO struct {
	MediaType   string     `json:"mediaType"`
	PresetID    string     `json:"presetId"`
	MatchMode   string     `json:"matchMode"`
	MatchTarget int        `json:"matchTarget,omitempty"`
	Threshold   int        `json:"threshold,omitempty"`
	DeckLimit   int        `json:"deckLimit"`
	Filters     FiltersDTO `json:"filters"`
}

type ParticipantDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar,omitempty"`
	IsHost   bool   `json:"isHost"`
	Online   bool   `json:"online"`
	Progress int    `json:"progress"`
	Done     bool   `json:"done"`
}

type MatchDTO struct {
	Movie     MovieDTO          `json:"movie"`
	MatchedAt time.Time         `json:"matchedAt"`
	Votes     map[string]string `json:"votes,omitempty"`
}

type NearMatchDTO struct {
	Movie  MovieDTO          `json:"movie"`
	Likes  int               `json:"likes"`
	Needed int               `json:"needed"`
	Votes  map[string]string `json:"votes,omitempty"`
}

type ParticipantStatsDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Likes int    `json:"likes"`
	Seen  int    `json:"seen"`
}

type ResultsDTO struct {
	Matches     []MatchDTO            `json:"matches"`
	NearMatches []NearMatchDTO        `json:"nearMatches"`
	Stats       []ParticipantStatsDTO `json:"stats"`
}

type ContinueDTO struct {
	ProposedLimit int             `json:"proposedLimit"`
	Votes         map[string]bool `json:"votes"`
}

type StateDTO struct {
	Code            string           `json:"code"`
	Status          string           `json:"status"`
	Round           int              `json:"round"`
	Settings        SettingsDTO      `json:"settings"`
	Participants    []ParticipantDTO `json:"participants"`
	Departed        []ParticipantDTO `json:"departed,omitempty"`
	DeckSize        int              `json:"deckSize"`
	SourceExhausted bool             `json:"sourceExhausted"`
	AllDone         bool             `json:"allDone"`
	NearMissCount   int              `json:"nearMissCount"`
	Matches         []MatchDTO       `json:"matches"`
	Requested       []int64          `json:"requested,omitempty"`
	Continue        *ContinueDTO     `json:"continue,omitempty"`
	Results         *ResultsDTO      `json:"results,omitempty"`
}

type YouDTO struct {
	ID     string `json:"id"`
	IsHost bool   `json:"isHost"`
	Cursor int    `json:"cursor"`
}

type PersonalStateDTO struct {
	State StateDTO `json:"state"`
	You   YouDTO   `json:"you"`
}

type CredentialsDTO struct {
	Code  string `json:"code"`
	Token string `json:"token"`
	You   YouDTO `json:"you"`
}

type DeckDTO struct {
	Movies []MovieDTO `json:"movies"`
}

type ProfileDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ProfilesDTO struct {
	Profiles  []ProfileDTO `json:"profiles"`
	DefaultID int          `json:"defaultId"`
}

type ConfigDTO struct {
	RequestsEnabled bool `json:"requestsEnabled"`
}
