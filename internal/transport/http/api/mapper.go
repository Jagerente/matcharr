package api

import (
	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/session"
)

func MapMovie(m domain.Movie) MovieDTO {
	return MovieDTO{
		ID:            m.ID,
		Title:         m.Title,
		OriginalTitle: m.OriginalTitle,
		Year:          m.Year,
		Genres:        m.Genres,
		Rating:        m.Rating,
		VoteCount:     m.VoteCount,
		Overview:      m.Overview,
		PosterPath:    m.PosterPath,
		BackdropPath:  m.BackdropPath,
		Runtime:       m.Runtime,
	}
}

func mapMovies(movies []domain.Movie) []MovieDTO {
	out := make([]MovieDTO, 0, len(movies))
	for _, m := range movies {
		out = append(out, MapMovie(m))
	}
	return out
}

func mapGenres(genres []domain.Genre) []GenreDTO {
	out := make([]GenreDTO, 0, len(genres))
	for _, g := range genres {
		out = append(out, GenreDTO{ID: g.ID, Name: g.Name})
	}
	return out
}

func mapPresets(presets []domain.Preset) []PresetDTO {
	out := make([]PresetDTO, 0, len(presets))
	for _, p := range presets {
		out = append(out, PresetDTO{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			Icon:        p.Icon,
			MediaType:   string(p.MediaType),
		})
	}
	return out
}

func mapSettings(s domain.Settings) SettingsDTO {
	return SettingsDTO{
		MediaType:   string(s.MediaType),
		PresetID:    s.PresetID,
		MatchMode:   string(s.MatchMode),
		MatchTarget: s.MatchTarget,
		Threshold:   s.Threshold,
		DeckLimit:   s.DeckLimit,
		Filters: FiltersDTO{
			GenreIDs:  s.Filters.GenreIDs,
			YearFrom:  s.Filters.YearFrom,
			YearTo:    s.Filters.YearTo,
			RatingMin: s.Filters.RatingMin,
			VotesMin:  s.Filters.VotesMin,
		},
	}
}

func settingsFromDTO(dto SettingsDTO) domain.Settings {
	return domain.Settings{
		MediaType:   domain.MediaType(dto.MediaType),
		PresetID:    dto.PresetID,
		MatchMode:   domain.MatchMode(dto.MatchMode),
		MatchTarget: dto.MatchTarget,
		Threshold:   dto.Threshold,
		DeckLimit:   dto.DeckLimit,
		Filters: domain.Filters{
			GenreIDs:  dto.Filters.GenreIDs,
			YearFrom:  dto.Filters.YearFrom,
			YearTo:    dto.Filters.YearTo,
			RatingMin: dto.Filters.RatingMin,
			VotesMin:  dto.Filters.VotesMin,
		},
	}
}

func mapVotes(votes map[string]domain.Verdict) map[string]string {
	if len(votes) == 0 {
		return nil
	}
	out := make(map[string]string, len(votes))
	for id, v := range votes {
		out[id] = string(v)
	}
	return out
}

func mapMatches(matches []domain.Match) []MatchDTO {
	out := make([]MatchDTO, 0, len(matches))
	for _, m := range matches {
		out = append(out, MatchDTO{
			Movie:     MapMovie(m.Movie),
			MatchedAt: m.MatchedAt,
			Votes:     mapVotes(m.Votes),
		})
	}
	return out
}

func mapResults(r *domain.Results) *ResultsDTO {
	if r == nil {
		return nil
	}
	near := make([]NearMatchDTO, 0, len(r.NearMatches))
	for _, n := range r.NearMatches {
		near = append(near, NearMatchDTO{
			Movie:  MapMovie(n.Movie),
			Likes:  n.Likes,
			Needed: n.Needed,
			Votes:  mapVotes(n.Votes),
		})
	}
	stats := make([]ParticipantStatsDTO, 0, len(r.Stats))
	for _, s := range r.Stats {
		stats = append(stats, ParticipantStatsDTO{ID: s.ID, Name: s.Name, Likes: s.Likes, Seen: s.Seen})
	}
	return &ResultsDTO{Matches: mapMatches(r.Matches), NearMatches: near, Stats: stats}
}

func MapState(view session.StateView) StateDTO {
	participants := make([]ParticipantDTO, 0, len(view.Participants))
	for _, p := range view.Participants {
		participants = append(participants, ParticipantDTO{
			ID:       p.ID,
			Name:     p.Name,
			Avatar:   p.Avatar,
			IsHost:   p.IsHost,
			Online:   p.Online,
			Progress: p.Progress,
			Done:     p.Done,
		})
	}

	var departed []ParticipantDTO
	for _, d := range view.Departed {
		departed = append(departed, ParticipantDTO{
			ID:       d.ID,
			Name:     d.Name,
			Avatar:   d.Avatar,
			Progress: d.Progress,
		})
	}

	var voting *ContinueDTO
	if view.Continue != nil {
		voting = &ContinueDTO{ProposedLimit: view.Continue.ProposedLimit, Votes: view.Continue.Votes}
	}

	return StateDTO{
		Code:            view.Code,
		Status:          string(view.Status),
		Round:           view.Round,
		Settings:        mapSettings(view.Settings),
		Participants:    participants,
		Departed:        departed,
		DeckSize:        view.DeckSize,
		SourceExhausted: view.SourceExhausted,
		AllDone:         view.AllDone,
		NearMissCount:   view.NearMissCount,
		Matches:         mapMatches(view.Matches),
		Requested:       view.Requested,
		Continue:        voting,
		Results:         mapResults(view.Results),
	}
}

func MapYou(y session.You) YouDTO {
	return YouDTO{ID: y.ID, IsHost: y.IsHost, Cursor: y.Cursor}
}

func mapCredentials(c session.Credentials) CredentialsDTO {
	return CredentialsDTO{Code: c.Code, Token: c.Token, You: MapYou(c.You)}
}

func mapProfiles(p domain.QualityProfiles) ProfilesDTO {
	profiles := make([]ProfileDTO, 0, len(p.Profiles))
	for _, profile := range p.Profiles {
		profiles = append(profiles, ProfileDTO{ID: profile.ID, Name: profile.Name})
	}
	return ProfilesDTO{Profiles: profiles, DefaultID: p.DefaultID}
}
