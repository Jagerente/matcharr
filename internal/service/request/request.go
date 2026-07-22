package request

import (
	"context"
	"log/slog"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

type Requester interface {
	Profiles(ctx context.Context) (domain.QualityProfiles, error)
	Request(ctx context.Context, movieID int64, profileID int) error
}

type SessionAuthority interface {
	AuthorizeHost(code, token string) error
	AuthorizeRequest(code, token string, movieID int64) error
	MarkRequested(code, token string, movieID int64) error
}

// Service treats a nil requester as an unconfigured integration; every
// call then reports it as disabled.
type Service struct {
	sessions  SessionAuthority
	requester Requester
	logger    *slog.Logger
}

func New(sessions SessionAuthority, requester Requester, logger *slog.Logger) *Service {
	return &Service{sessions: sessions, requester: requester, logger: logger}
}

func (s *Service) Enabled() bool { return s.requester != nil }

func (s *Service) Profiles(ctx context.Context, code, token string) (domain.QualityProfiles, error) {
	if !s.Enabled() {
		return domain.QualityProfiles{}, errDisabled()
	}
	if err := s.sessions.AuthorizeHost(code, token); err != nil {
		return domain.QualityProfiles{}, err
	}
	profiles, err := s.requester.Profiles(ctx)
	if err != nil {
		return domain.QualityProfiles{}, apperr.Wrap(err, apperr.KindUpstream, apperr.ReasonProfilesFailed, "could not load the quality profiles")
	}
	return profiles, nil
}

func (s *Service) Submit(ctx context.Context, code, token string, movieID int64, profileID int) error {
	if !s.Enabled() {
		return errDisabled()
	}
	if err := s.sessions.AuthorizeRequest(code, token, movieID); err != nil {
		return err
	}
	if err := s.requester.Request(ctx, movieID, profileID); err != nil {
		return apperr.Wrap(err, apperr.KindUpstream, apperr.ReasonRequestFailed, "could not submit the request")
	}
	if err := s.sessions.MarkRequested(code, token, movieID); err != nil {
		// The request already went out; failing the call now would make the
		// host retry a submission that succeeded.
		s.logger.Error("marking the request", "movie", movieID, "err", err)
	}
	return nil
}

func errDisabled() error {
	return apperr.New(apperr.KindNotImplemented, apperr.ReasonRequestsDisabled, "the request integration is not configured")
}
