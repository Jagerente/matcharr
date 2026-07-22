<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { imageUrl, POSTER_FULL } from '../api/client'
import type { Movie, Verdict } from '../api/client'
import { t } from '../i18n'
import * as store from '../store/session'
import Sheet from './Sheet.vue'
import QualityPicker from './QualityPicker.vue'
import Spinner from './Spinner.vue'
import Icon from './Icon.vue'

const props = defineProps<{
  movie: Movie
  votes?: Record<string, Verdict>
}>()
const emit = defineEmits<{ close: [] }>()

const full = ref<Movie | null>(null)
const shown = computed(() => full.value ?? props.movie)
const failed = ref(false)
const sending = ref(false)
const requested = computed(() => store.isRequested(props.movie.id))

onMounted(async () => {
  full.value = await store.movieDetails(props.movie.id)
})

interface VoteRow {
  id: string
  name: string
  avatar: string
  verdict: 'like' | 'nope' | 'unseen'
}

const rows = computed<VoteRow[]>(() => {
  const s = store.state.value
  if (!s) return []
  const everyone = [...s.participants, ...(s.departed ?? [])]
  return everyone.map((p) => ({
    id: p.id,
    name: p.name,
    avatar: p.avatar || '🙂',
    verdict: props.votes?.[p.id] ?? 'unseen',
  }))
})

async function request() {
  sending.value = true
  failed.value = false
  const ok = await store.requestMovie(props.movie.id)
  sending.value = false
  failed.value = !ok
}
</script>

<template>
  <Sheet class="detail-sheet" :layer="70" @close="emit('close')">
    <div class="detail-sheet__scroll">
        <div class="detail-sheet__head">
          <img
            v-if="movie.posterPath"
            class="detail-sheet__poster"
            :src="imageUrl(movie.posterPath, POSTER_FULL)"
            :alt="movie.title"
          />
          <div v-else class="detail-sheet__poster poster-fallback"></div>
          <div>
            <h2 class="detail-sheet__title">{{ shown.title }}</h2>
            <div
              v-if="shown.originalTitle && shown.originalTitle !== shown.title"
              class="detail-sheet__orig"
            >
              {{ shown.originalTitle }}
            </div>
            <div class="detail-sheet__facts">
              <span v-if="shown.year">{{ shown.year }}</span>
              <span v-if="shown.rating" class="detail-sheet__star">★ {{ shown.rating.toFixed(1) }}</span>
              <span v-if="shown.runtime">{{ t('movie.runtime', { count: shown.runtime }) }}</span>
            </div>
          </div>
        </div>

        <p class="detail-sheet__overview">{{ shown.overview || t('movie.noOverview') }}</p>

        <div v-if="rows.length" class="label detail-sheet__label">{{ t('detail.votes') }}</div>
        <div class="detail-sheet__votes">
          <div v-for="r in rows" :key="r.id" class="detail-sheet__vote">
            <span class="detail-sheet__avatar">{{ r.avatar }}</span>
            <span class="detail-sheet__name">{{ r.name }}</span>
            <span v-if="r.verdict === 'like'" class="detail-sheet__liked">
              <Icon name="heart" filled :size="12" /> {{ t('detail.liked') }}
            </span>
            <span v-else-if="r.verdict === 'nope'" class="detail-sheet__passed">
              <Icon name="x" :size="12" /> {{ t('detail.passed') }}
            </span>
            <span v-else class="detail-sheet__unseen">{{ t('detail.unseen') }}</span>
          </div>
        </div>
      </div>

    <div v-if="store.canRequest.value" class="detail-sheet__foot">
      <div v-if="failed" class="detail-sheet__fail" @click="request">
        <Icon name="alert" :size="15" class="detail-sheet__fail-icon" />
        {{ t('detail.requestFailed') }}
      </div>
      <QualityPicker v-if="!requested" :disabled="sending" />
      <button
        class="btn btn--primary"
        type="button"
        :disabled="requested || sending"
        @click="request"
      >
        <Spinner v-if="sending" />
        <template v-else-if="requested">
          {{ t('detail.requested') }} <Icon name="check" :size="16" />
        </template>
        <template v-else>{{ t('detail.request') }} <Icon name="download" :size="16" /></template>
      </button>
    </div>
  </Sheet>
</template>

<style>
.detail-sheet {
  display: flex;
  flex-direction: column;
  max-height: 88dvh;
}

.detail-sheet__scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.detail-sheet__head {
  display: flex;
  gap: 14px;
  margin-bottom: 16px;
}

.detail-sheet__poster {
  width: 78px;
  aspect-ratio: 2 / 3;
  border-radius: var(--radius-s);
  object-fit: cover;
  flex: none;
  border: 1px solid var(--border);
}

.detail-sheet__title {
  font-size: var(--fs-4);
  font-weight: 700;
  margin: 0 0 2px;
  line-height: 1.1;
}

.detail-sheet__orig {
  font-size: var(--fs-1);
  color: var(--text-3);
  font-style: italic;
  margin-bottom: 6px;
}

.detail-sheet__facts {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--fs-2);
  color: var(--text-on-image);
}

.detail-sheet__star {
  color: var(--star);
}

.detail-sheet__overview {
  font-size: var(--fs-3);
  line-height: 1.55;
  color: var(--text-body);
  margin: 0 0 18px;
}

.detail-sheet__label {
  margin-bottom: 10px;
}

.detail-sheet__votes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 8px;
}

.detail-sheet__vote {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--raised);
  border-radius: var(--radius-s);
  padding: 8px 12px;
}

.detail-sheet__avatar {
  font-size: 20px;
}

.detail-sheet__name {
  font-size: var(--fs-2);
  flex: 1;
}

.detail-sheet__liked {
  font-size: var(--fs-2);
  color: var(--online);
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 5px;
}

.detail-sheet__passed {
  font-size: var(--fs-2);
  color: var(--text-3);
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 5px;
}

.detail-sheet__unseen {
  font-size: var(--fs-2);
  color: var(--text-3);
}

.detail-sheet__foot {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-top: 12px;
}

.detail-sheet__fail {
  background: var(--raised);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-m);
  padding: 11px 14px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: var(--fs-2);
  color: var(--text);
  cursor: pointer;
}

.detail-sheet__fail-icon {
  color: var(--accent);
}
</style>
