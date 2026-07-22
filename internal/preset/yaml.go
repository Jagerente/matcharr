package preset

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Jagerente/matcharr/internal/domain"
)

type overlayFile struct {
	Presets []overlayPreset `yaml:"presets"`
}

type overlayPreset struct {
	ID          string            `yaml:"id"`
	Title       map[string]string `yaml:"title"`
	Description map[string]string `yaml:"description"`
	Icon        string            `yaml:"icon"`
	MediaType   string            `yaml:"mediaType"`
	Source      string            `yaml:"source"`
	Endpoint    string            `yaml:"endpoint"`
	Params      map[string]any    `yaml:"params"`
	Order       int               `yaml:"order"`
	Hidden      bool              `yaml:"hidden"`
}

// LoadOverlay treats a missing file as normal and any other error as
// fatal, so a yaml typo does not hide behind a silent fallback.
func LoadOverlay(path string) ([]domain.Preset, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var file overlayFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	out := make([]domain.Preset, 0, len(file.Presets))
	for i, p := range file.Presets {
		preset, err := p.toDomain(i)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, preset)
	}
	return out, nil
}

func (p overlayPreset) toDomain(idx int) (domain.Preset, error) {
	if p.ID == "" {
		return domain.Preset{}, fmt.Errorf("preset #%d: missing id", idx+1)
	}
	if !p.Hidden && p.Source == "" {
		return domain.Preset{}, fmt.Errorf("preset %q: missing source", p.ID)
	}

	params, err := stringifyParams(p.Params)
	if err != nil {
		return domain.Preset{}, fmt.Errorf("preset %q: %w", p.ID, err)
	}

	mediaType := domain.MediaType(p.MediaType)
	if mediaType == "" {
		mediaType = domain.MediaTypeMovie
	}

	order := p.Order
	if order == 0 {
		// User presets without an explicit order sort after built-ins.
		order = 1000 + idx
	}

	return domain.Preset{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Icon:        p.Icon,
		MediaType:   mediaType,
		Source:      p.Source,
		Endpoint:    p.Endpoint,
		Params:      params,
		Order:       order,
		Hidden:      p.Hidden,
	}, nil
}

func stringifyParams(in map[string]any) (map[string]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		switch v.(type) {
		case string, int, int64, uint64, float32, float64, bool:
			out[k] = fmt.Sprint(v)
		default:
			return nil, fmt.Errorf("param %q: expected a scalar value, got %T", k, v)
		}
	}
	return out, nil
}
