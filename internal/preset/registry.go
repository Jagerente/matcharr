package preset

import (
	"sort"

	"github.com/Jagerente/matcharr/internal/domain"
)

// Registry is immutable after initialization, so no locking.
type Registry struct {
	order   []string
	presets map[string]domain.Preset
}

func NewRegistry(presets []domain.Preset) *Registry {
	r := &Registry{presets: make(map[string]domain.Preset, len(presets))}
	for _, p := range presets {
		r.add(p)
	}
	return r
}

// Apply lets an id match fully replace the built-in preset.
func (r *Registry) Apply(overlay []domain.Preset) {
	for _, p := range overlay {
		r.add(p)
	}
}

func (r *Registry) add(p domain.Preset) {
	if _, exists := r.presets[p.ID]; !exists {
		r.order = append(r.order, p.ID)
	}
	r.presets[p.ID] = p
}

func (r *Registry) List() []domain.Preset {
	out := make([]domain.Preset, 0, len(r.order))
	for _, id := range r.order {
		if p := r.presets[id]; !p.Hidden {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

func (r *Registry) Get(id string) (domain.Preset, bool) {
	p, ok := r.presets[id]
	if !ok || p.Hidden {
		return domain.Preset{}, false
	}
	return p, true
}
