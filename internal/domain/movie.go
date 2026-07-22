package domain

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
)

type Movie struct {
	ID            int64
	Title         string
	OriginalTitle string
	Year          int
	Genres        []string
	Rating        float64
	VoteCount     int
	Overview      string
	PosterPath    string
	BackdropPath  string
	Runtime       int
}
