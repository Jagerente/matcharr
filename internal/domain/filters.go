package domain

type Filters struct {
	GenreIDs  []int64
	YearFrom  int
	YearTo    int
	RatingMin float64
	VotesMin  int
}

func (f Filters) IsZero() bool {
	return len(f.GenreIDs) == 0 && f.YearFrom == 0 && f.YearTo == 0 &&
		f.RatingMin == 0 && f.VotesMin == 0
}
