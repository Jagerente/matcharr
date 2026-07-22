package preset

import (
	"github.com/Jagerente/matcharr/internal/client/tmdb"
	"github.com/Jagerente/matcharr/internal/domain"
)

func Builtins() []domain.Preset {
	return []domain.Preset{
		{
			ID:        "trending-week",
			Title:     map[string]string{"en": "Trending this week", "ru": "Популярное за неделю", "zh-Hans": "本周热门"},
			Icon:      "🔥",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointTrending,
			Params:    map[string]string{"window": "week"},
			Order:     10,
		},
		{
			ID:        "popular-now",
			Title:     map[string]string{"en": "Popular now", "ru": "Популярное сейчас", "zh-Hans": "当前热门"},
			Icon:      "⭐",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointPopular,
			Order:     20,
		},
		{
			ID:        "all-time-popular",
			Title:     map[string]string{"en": "All-time popular", "ru": "Популярное за всё время", "zh-Hans": "历久不衰"},
			Icon:      "🏆",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointDiscover,
			Params:    map[string]string{"sort_by": "vote_count.desc"},
			Order:     30,
		},
		{
			ID:        "top-rated",
			Title:     map[string]string{"en": "Top rated", "ru": "Высший рейтинг", "zh-Hans": "评分最高"},
			Icon:      "🎯",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointDiscover,
			Params: map[string]string{
				"sort_by":        "vote_average.desc",
				"vote_count.gte": "2000",
			},
			Order: 40,
		},
		{
			ID:        "new-releases",
			Title:     map[string]string{"en": "New releases", "ru": "Новинки", "zh-Hans": "新片上映"},
			Icon:      "🆕",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointDiscover,
			Params: map[string]string{
				"primary_release_date.gte": "now-90d",
				"primary_release_date.lte": "now",
				"sort_by":                  "popularity.desc",
			},
			Order: 50,
		},
		{
			ID:        "now-playing",
			Title:     map[string]string{"en": "Now in theaters", "ru": "Сейчас в кино", "zh-Hans": "正在影院"},
			Icon:      "🎬",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointNowPlaying,
			Order:     60,
		},
		{
			ID:    "custom",
			Title: map[string]string{"en": "Custom", "ru": "Свой набор", "zh-Hans": "自定义"},
			Description: map[string]string{
				"en":      "Empty preset: the deck is built from filters only",
				"ru":      "Пустой пресет: колода собирается только по фильтрам",
				"zh-Hans": "空预设：牌堆仅由筛选条件生成",
			},
			Icon:      "🎛️",
			MediaType: domain.MediaTypeMovie,
			Source:    tmdb.SourceID,
			Endpoint:  tmdb.EndpointDiscover,
			Order:     70,
		},
	}
}
