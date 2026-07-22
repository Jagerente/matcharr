package deck

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/Jagerente/matcharr/internal/domain"
)

type fakePresets map[string]domain.Preset

func (f fakePresets) Get(id string) (domain.Preset, bool) {
	p, ok := f[id]
	return p, ok
}

type fakeSource struct {
	pages      map[int][]domain.Movie
	totalPages int
	failPage   int
	calls      int
}

func (f *fakeSource) ID() string { return "fake" }

func (f *fakeSource) FetchPage(_ context.Context, _ Query, page int) (Page, error) {
	f.calls++
	if f.failPage != 0 && page == f.failPage {
		return Page{}, errors.New("upstream down")
	}
	return Page{Movies: f.pages[page], Page: page, TotalPages: f.totalPages}, nil
}

func movies(ids ...int64) []domain.Movie {
	out := make([]domain.Movie, 0, len(ids))
	for _, id := range ids {
		out = append(out, domain.Movie{ID: id})
	}
	return out
}

func newBuilder(t *testing.T, src *fakeSource) *Builder {
	t.Helper()
	presets := fakePresets{"p": {ID: "p", Source: "fake"}}
	return NewBuilder(presets, NewRegistry(src), slog.New(slog.DiscardHandler))
}

func ids(result Result) []int64 {
	out := make([]int64, 0, len(result.Movies))
	for _, m := range result.Movies {
		out = append(out, m.ID)
	}
	return out
}

func TestBuildPagesUntilFilled(t *testing.T) {
	src := &fakeSource{pages: map[int][]domain.Movie{
		1: movies(1, 2),
		2: movies(2, 3, 4),
	}, totalPages: 9}

	result, err := newBuilder(t, src).Build(context.Background(), Request{PresetID: "p", StartPage: 1, Want: 4})
	if err != nil {
		t.Fatal(err)
	}
	got := ids(result)
	if len(got) != 4 || got[0] != 1 || got[3] != 4 {
		t.Fatalf("Build() = %v, want 1..4 with the duplicate dropped", got)
	}
	if result.NextPage != 3 || result.Exhausted {
		t.Fatalf("NextPage = %d, Exhausted = %v", result.NextPage, result.Exhausted)
	}
}

func TestBuildKeepsUnfinishedPage(t *testing.T) {
	src := &fakeSource{pages: map[int][]domain.Movie{1: movies(1, 2, 3)}, totalPages: 9}

	result, err := newBuilder(t, src).Build(context.Background(), Request{PresetID: "p", StartPage: 1, Want: 2})
	if err != nil {
		t.Fatal(err)
	}
	// The limit filled mid-page: the page is not turned so the next round
	// picks up what is left on it.
	if result.NextPage != 1 {
		t.Fatalf("NextPage = %d, want the unfinished page 1", result.NextPage)
	}
}

func TestBuildExcludes(t *testing.T) {
	src := &fakeSource{pages: map[int][]domain.Movie{1: movies(1, 2, 3)}, totalPages: 1}

	result, err := newBuilder(t, src).Build(context.Background(), Request{
		PresetID:  "p",
		StartPage: 1,
		Want:      3,
		Exclude:   map[int64]struct{}{2: {}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := ids(result)
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("Build() = %v, want the excluded movie skipped", got)
	}
	if !result.Exhausted {
		t.Fatal("a single-page source must report exhaustion")
	}
}

func TestBuildEmptyPageMeansExhausted(t *testing.T) {
	src := &fakeSource{pages: map[int][]domain.Movie{}}

	result, err := newBuilder(t, src).Build(context.Background(), Request{PresetID: "p", StartPage: 1, Want: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Movies) != 0 || !result.Exhausted {
		t.Fatalf("Build() = %v exhausted %v, want empty and exhausted", result.Movies, result.Exhausted)
	}
}

func TestBuildPartialBeatsFailure(t *testing.T) {
	src := &fakeSource{pages: map[int][]domain.Movie{1: movies(1)}, failPage: 2, totalPages: 9}

	result, err := newBuilder(t, src).Build(context.Background(), Request{PresetID: "p", StartPage: 1, Want: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Movies) != 1 || result.Exhausted {
		t.Fatalf("Build() = %v, want the partial page kept", result.Movies)
	}
}

func TestBuildFirstPageFailureIsFatal(t *testing.T) {
	src := &fakeSource{failPage: 1}

	if _, err := newBuilder(t, src).Build(context.Background(), Request{PresetID: "p", StartPage: 1, Want: 5}); err == nil {
		t.Fatal("Build() with nothing collected must fail")
	}
}

func TestBuildUnknownPresetAndSource(t *testing.T) {
	src := &fakeSource{}
	builder := NewBuilder(fakePresets{"orphan": {ID: "orphan", Source: "missing"}}, NewRegistry(src), slog.New(slog.DiscardHandler))

	if _, err := builder.Build(context.Background(), Request{PresetID: "nope"}); err == nil {
		t.Fatal("unknown preset must fail")
	}
	if _, err := builder.Build(context.Background(), Request{PresetID: "orphan"}); err == nil {
		t.Fatal("unwired source must fail")
	}
	if builder.HasPreset("nope") || !builder.HasPreset("orphan") {
		t.Fatal("HasPreset() disagrees with the catalog")
	}
}
