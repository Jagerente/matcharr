package session

import (
	"slices"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

func (s *Service) State(code, token string) (PersonalView, error) {
	var view PersonalView
	err := s.exec(code, require{token: token}, func(tx *Tx) error {
		view = PersonalView{State: buildView(tx.Session()), You: you(tx.Actor())}
		return nil
	})
	return view, err
}

func (s *Service) Deck(code, token string) ([]domain.Movie, error) {
	var movies []domain.Movie
	err := s.exec(code, require{token: token, statuses: afterLobby}, func(tx *Tx) error {
		movies = slices.Clone(tx.Session().Deck)
		return nil
	})
	return movies, err
}

func (s *Service) Authenticate(code, token string) (You, error) {
	var who You
	err := s.exec(code, require{token: token}, func(tx *Tx) error {
		who = you(tx.Actor())
		return nil
	})
	return who, err
}

func (s *Service) AuthorizeHost(code, token string) error {
	return s.exec(code, require{token: token, host: true}, func(*Tx) error {
		return nil
	})
}

// AuthorizeMovie confirms the caller belongs to the session and the movie
// is one this session actually dealt, so no endpoint keyed by catalog id
// can be used to walk the catalog.
func (s *Service) AuthorizeMovie(code, token string, movieID int64) error {
	return s.exec(code, require{token: token}, func(tx *Tx) error {
		return requireInDeck(tx, movieID)
	})
}

// AuthorizeRequest guards the download request: the host decides, and only
// over what this session actually dealt, so the endpoint cannot become an
// open proxy into the download service.
func (s *Service) AuthorizeRequest(code, token string, movieID int64) error {
	return s.exec(code, require{token: token, host: true}, func(tx *Tx) error {
		return requireInDeck(tx, movieID)
	})
}

// MarkRequested records a movie the download service accepted, so every
// client (and a reloaded host) sees it as already sent.
func (s *Service) MarkRequested(code, token string, movieID int64) error {
	return s.exec(code, require{token: token, host: true}, func(tx *Tx) error {
		sess := tx.Session()
		if slices.Contains(sess.Requested, movieID) {
			return nil
		}
		sess.MarkRequested(movieID)
		tx.Emit(EventRequestSent, RequestSentData{MovieID: movieID})
		return nil
	})
}

func requireInDeck(tx *Tx, movieID int64) error {
	if !tx.Session().InDeck(movieID) {
		return invalid(apperr.ReasonUnknownMovie, "movie is not part of this session")
	}
	return nil
}
