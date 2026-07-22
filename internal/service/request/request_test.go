package request

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

type fakeAuthority struct {
	hostErr    error
	requestErr error
	marked     []int64
	markErr    error
}

func (f *fakeAuthority) AuthorizeHost(_, _ string) error { return f.hostErr }

func (f *fakeAuthority) AuthorizeRequest(_, _ string, _ int64) error { return f.requestErr }

func (f *fakeAuthority) MarkRequested(_, _ string, movieID int64) error {
	f.marked = append(f.marked, movieID)
	return f.markErr
}

type fakeRequester struct {
	profiles  domain.QualityProfiles
	err       error
	requested []int64
}

func (f *fakeRequester) Profiles(context.Context) (domain.QualityProfiles, error) {
	return f.profiles, f.err
}

func (f *fakeRequester) Request(_ context.Context, movieID int64, _ int) error {
	f.requested = append(f.requested, movieID)
	return f.err
}

func testService(authority *fakeAuthority, requester Requester) *Service {
	return New(authority, requester, slog.New(slog.DiscardHandler))
}

func wantReason(t *testing.T, err error, reason apperr.Reason) {
	t.Helper()
	appErr := apperr.From(err)
	if appErr == nil || appErr.Reason != reason {
		t.Fatalf("err = %v, want reason %q", err, reason)
	}
}

func TestDisabledIntegration(t *testing.T) {
	svc := testService(&fakeAuthority{}, nil)
	if svc.Enabled() {
		t.Fatal("Enabled() without a requester")
	}
	_, err := svc.Profiles(context.Background(), "AAAA", "t")
	wantReason(t, err, apperr.ReasonRequestsDisabled)
	wantReason(t, svc.Submit(context.Background(), "AAAA", "t", 1, 0), apperr.ReasonRequestsDisabled)
}

func TestProfilesHostOnly(t *testing.T) {
	hostErr := apperr.New(apperr.KindForbidden, apperr.ReasonHostOnly, "not the host")
	svc := testService(&fakeAuthority{hostErr: hostErr}, &fakeRequester{})
	_, err := svc.Profiles(context.Background(), "AAAA", "t")
	wantReason(t, err, apperr.ReasonHostOnly)
}

func TestProfilesUpstreamFailure(t *testing.T) {
	svc := testService(&fakeAuthority{}, &fakeRequester{err: errors.New("down")})
	_, err := svc.Profiles(context.Background(), "AAAA", "t")
	wantReason(t, err, apperr.ReasonProfilesFailed)
}

func TestSubmitMarksTheRequest(t *testing.T) {
	authority := &fakeAuthority{}
	requester := &fakeRequester{}
	svc := testService(authority, requester)

	if err := svc.Submit(context.Background(), "AAAA", "t", 7, 2); err != nil {
		t.Fatal(err)
	}
	if len(requester.requested) != 1 || requester.requested[0] != 7 {
		t.Fatalf("requested = %v", requester.requested)
	}
	if len(authority.marked) != 1 || authority.marked[0] != 7 {
		t.Fatalf("marked = %v", authority.marked)
	}
}

func TestSubmitFailureDoesNotMark(t *testing.T) {
	authority := &fakeAuthority{}
	svc := testService(authority, &fakeRequester{err: errors.New("down")})

	wantReason(t, svc.Submit(context.Background(), "AAAA", "t", 7, 0), apperr.ReasonRequestFailed)
	if len(authority.marked) != 0 {
		t.Fatal("a failed request must not be marked as sent")
	}
}

func TestSubmitSurvivesMarkFailure(t *testing.T) {
	authority := &fakeAuthority{markErr: errors.New("gone")}
	svc := testService(authority, &fakeRequester{})

	// The request already went out; a bookkeeping failure must not make
	// the host retry a submission that succeeded.
	if err := svc.Submit(context.Background(), "AAAA", "t", 7, 0); err != nil {
		t.Fatal(err)
	}
}
