package preset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Jagerente/matcharr/internal/domain"
)

func testPresets() []domain.Preset {
	return []domain.Preset{
		{ID: "second", Order: 20},
		{ID: "first", Order: 10},
	}
}

func TestRegistryListSortsByOrder(t *testing.T) {
	r := NewRegistry(testPresets())

	list := r.List()
	if len(list) != 2 || list[0].ID != "first" || list[1].ID != "second" {
		t.Fatalf("List() = %v, want order first, second", ids(list))
	}
}

func TestRegistryOverlayReplacesAndHides(t *testing.T) {
	r := NewRegistry(testPresets())
	r.Apply([]domain.Preset{
		{ID: "first", Icon: "🎃", Order: 10}, // full replacement of a built-in
		{ID: "second", Hidden: true},        // hiding a built-in
		{ID: "added", Order: 1000},          // a new user preset
	})

	list := r.List()
	if len(list) != 2 || list[0].ID != "first" || list[1].ID != "added" {
		t.Fatalf("List() = %v, want first, added", ids(list))
	}
	if list[0].Icon != "🎃" {
		t.Fatalf("overlay did not replace the preset: icon = %q", list[0].Icon)
	}

	if _, ok := r.Get("second"); ok {
		t.Fatal("Get() returned a hidden preset")
	}
}

func TestLoadOverlayMissingFile(t *testing.T) {
	overlay, err := LoadOverlay(filepath.Join(t.TempDir(), "presets.yml"))
	if err != nil {
		t.Fatalf("a missing file must be fine, got: %v", err)
	}
	if overlay != nil {
		t.Fatalf("want an empty overlay, got %v", overlay)
	}
}

func TestLoadOverlayParsesAndValidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.yml")
	writeFile(t, path, `
presets:
  - id: cozy-horror
    title: { en: "Cozy horrors" }
    icon: "🎃"
    source: tmdb
    endpoint: discover
    params:
      with_genres: 27
      vote_average.gte: 6.5
  - id: now-playing
    hidden: true
`)

	overlay, err := LoadOverlay(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(overlay) != 2 {
		t.Fatalf("want 2 presets, got %d", len(overlay))
	}

	p := overlay[0]
	if p.Params["with_genres"] != "27" || p.Params["vote_average.gte"] != "6.5" {
		t.Fatalf("numeric params were not stringified: %v", p.Params)
	}
	if p.MediaType != domain.MediaTypeMovie {
		t.Fatalf("default mediaType must be movie, got %q", p.MediaType)
	}
	if p.Order < 1000 {
		t.Fatalf("a user preset without order must go after built-ins, order = %d", p.Order)
	}

	if !overlay[1].Hidden {
		t.Fatal("the hidden entry lost its flag")
	}
}

func TestLoadOverlayRejectsMissingSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.yml")
	writeFile(t, path, "presets:\n  - id: broken\n")

	if _, err := LoadOverlay(path); err == nil {
		t.Fatal("a preset without a source must be an error")
	}
}

func ids(presets []domain.Preset) []string {
	out := make([]string, len(presets))
	for i, p := range presets {
		out[i] = p.ID
	}
	return out
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
