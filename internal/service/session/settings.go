package session

import (
	"strings"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

const (
	maxNameLength = 24
	// maxAvatarLength fits a single emoji, which can still be a long
	// sequence of joiners and modifiers.
	maxAvatarLength = 16
	maxDeviceLength = 64
	maxMatchTarget  = 20
	maxGenreFilters = 20
)

type identity struct {
	name     string
	avatar   string
	deviceID string
}

func validateIdentity(name, avatar, deviceID string) (identity, error) {
	runes := []rune(strings.TrimSpace(name))
	if len(runes) == 0 {
		return identity{}, invalid(apperr.ReasonNameRequired, "a name is required")
	}
	if len(runes) > maxNameLength {
		return identity{}, invalid(apperr.ReasonNameTooLong, "name longer than %d characters", maxNameLength)
	}
	if len([]rune(avatar)) > maxAvatarLength {
		return identity{}, invalid(apperr.ReasonBadRequest, "avatar longer than %d characters", maxAvatarLength)
	}
	if len(deviceID) > maxDeviceLength {
		return identity{}, invalid(apperr.ReasonBadRequest, "device id longer than %d characters", maxDeviceLength)
	}
	return identity{name: string(runes), avatar: avatar, deviceID: deviceID}, nil
}

func (s *Service) normalizeSettings(set *domain.Settings) error {
	if set.MediaType == "" {
		set.MediaType = domain.MediaTypeMovie
	}
	if set.MediaType != domain.MediaTypeMovie {
		return invalid(apperr.ReasonBadSettings, "unsupported media type %q", set.MediaType)
	}
	if !s.decks.HasPreset(set.PresetID) {
		return invalid(apperr.ReasonUnknownPreset, "unknown preset %q", set.PresetID)
	}
	switch set.MatchMode {
	case "":
		set.MatchMode = domain.MatchModeFirst
	case domain.MatchModeFirst, domain.MatchModeAll:
	case domain.MatchModeCount:
		if set.MatchTarget < 1 || set.MatchTarget > maxMatchTarget {
			return invalid(apperr.ReasonBadSettings, "match count out of range")
		}
	default:
		return invalid(apperr.ReasonBadSettings, "unknown match mode %q", set.MatchMode)
	}
	if set.DeckLimit == 0 {
		set.DeckLimit = s.limits.defaultDeck()
	}
	low, high := s.limits.deckRange()
	if set.DeckLimit < low || set.DeckLimit > high {
		return invalid(apperr.ReasonBadSettings, "deck limit out of the %d..%d range", low, high)
	}
	if set.Threshold != 0 && (set.Threshold < 2 || set.Threshold > s.limits.MaxParticipants) {
		return invalid(apperr.ReasonBadSettings, "match threshold out of the 2..%d range", s.limits.MaxParticipants)
	}
	return validateFilters(set.Filters)
}

func validateFilters(f domain.Filters) error {
	for _, year := range []int{f.YearFrom, f.YearTo} {
		if year != 0 && (year < 1870 || year > 2100) {
			return invalid(apperr.ReasonBadSettings, "filter year out of range")
		}
	}
	if f.YearFrom != 0 && f.YearTo != 0 && f.YearFrom > f.YearTo {
		return invalid(apperr.ReasonBadSettings, "inverted year range")
	}
	if f.RatingMin < 0 || f.RatingMin > 10 {
		return invalid(apperr.ReasonBadSettings, "minimum rating out of range")
	}
	if f.VotesMin < 0 {
		return invalid(apperr.ReasonBadSettings, "negative minimum vote count")
	}
	if len(f.GenreIDs) > maxGenreFilters {
		return invalid(apperr.ReasonBadSettings, "too many genres selected")
	}
	return nil
}
