package tmdb

import (
	"slices"
	"strconv"

	"github.com/Jagerente/matcharr/internal/domain"
)

func toDomain(dto movieDTO, genres map[int64]string) domain.Movie {
	m := domain.Movie{
		ID:            dto.ID,
		Title:         dto.Title,
		OriginalTitle: dto.OriginalTitle,
		Year:          releaseYear(dto.ReleaseDate),
		Rating:        dto.VoteAverage,
		VoteCount:     dto.VoteCount,
		Overview:      dto.Overview,
		PosterPath:    dto.PosterPath,
		BackdropPath:  dto.BackdropPath,
	}
	for _, id := range dto.GenreIDs {
		if name, ok := genres[id]; ok {
			m.Genres = append(m.Genres, name)
		}
	}
	return m
}

func detailsToDomain(dto detailsDTO) domain.Movie {
	m := domain.Movie{
		ID:            dto.ID,
		Title:         dto.Title,
		OriginalTitle: dto.OriginalTitle,
		Year:          releaseYear(dto.ReleaseDate),
		Runtime:       dto.Runtime,
		Rating:        dto.VoteAverage,
		VoteCount:     dto.VoteCount,
		Overview:      dto.Overview,
		PosterPath:    dto.PosterPath,
		BackdropPath:  dto.BackdropPath,
	}
	for _, g := range dto.Genres {
		m.Genres = append(m.Genres, g.Name)
	}
	return m
}

func dtoMatches(dto movieDTO, f domain.Filters) bool {
	if len(f.GenreIDs) > 0 && !hasAnyGenre(dto.GenreIDs, f.GenreIDs) {
		return false
	}

	if f.YearFrom > 0 || f.YearTo > 0 {
		year := releaseYear(dto.ReleaseDate)
		if year == 0 {
			return false
		}
		if f.YearFrom > 0 && year < f.YearFrom {
			return false
		}
		if f.YearTo > 0 && year > f.YearTo {
			return false
		}
	}

	if f.RatingMin > 0 && dto.VoteAverage < f.RatingMin {
		return false
	}
	if f.VotesMin > 0 && dto.VoteCount < f.VotesMin {
		return false
	}
	return true
}

func hasAnyGenre(have, want []int64) bool {
	for _, id := range want {
		if slices.Contains(have, id) {
			return true
		}
	}
	return false
}

func releaseYear(releaseDate string) int {
	if len(releaseDate) < 4 {
		return 0
	}
	year, err := strconv.Atoi(releaseDate[:4])
	if err != nil {
		return 0
	}
	return year
}
