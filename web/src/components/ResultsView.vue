<script setup lang="ts">
import { computed, ref } from 'vue'
import { imageUrl, POSTER_FULL, POSTER_THUMB } from '../api/client'
import type { Match, NearMatch, Verdict } from '../api/client'
import * as store from '../store/session'
import { useBackGuard } from '../nav'
import { t } from '../i18n'
import ConfirmSheet from './ConfirmSheet.vue'
import DetailSheet from './DetailSheet.vue'
import QualityPicker from './QualityPicker.vue'
import AppHeader from './AppHeader.vue'
import Spinner from './Spinner.vue'
import Icon from './Icon.vue'

const state = computed(() => store.state.value!)
const results = computed(() => state.value.results)
const matches = computed(() => results.value?.matches ?? [])
const near = computed(() => results.value?.nearMatches ?? [])
const single = computed(() => state.value.settings.matchMode === 'first' && matches.value.length === 1)

const canRequest = store.canRequest
const opened = ref<{ movie: Match['movie']; votes?: Record<string, Verdict> } | null>(null)
const leaveOpen = ref(false)
const sending = ref(false)

useBackGuard(() => {
  leaveOpen.value = true
})
const tapHint = computed(() =>
  canRequest.value ? t('results.tapDetailsRequest') : t('results.tapDetails'),
)

function holdOut(n: NearMatch): string {
  const names = state.value.participants
    .filter((p) => n.votes?.[p.id] !== 'like')
    .map((p) => p.name)
  if (names.length === 1) return t('results.holdOutOne', { name: names[0]! })
  if (names.length > 1) return t('results.holdOutMany', { likes: n.likes, names: names.join(', ') })
  return t('results.holdOutShort', { count: n.likes })
}

function open(m: Match | NearMatch) {
  opened.value = { movie: m.movie, votes: m.votes }
}

async function requestWinner() {
  const m = matches.value[0]
  if (!m) return
  sending.value = true
  try {
    await store.requestMovie(m.movie.id)
  } finally {
    sending.value = false
  }
}

async function confirmLeave() {
  leaveOpen.value = false
  await store.leave()
}
</script>

<template>
  <div class="results">
    <AppHeader />
    <div class="results__scroll">
      <div v-if="!matches.length" class="results__hero-empty">
        <div class="hero-glyph">🤷</div>
        <h1 class="results__title">{{ t('results.emptyTitle') }}</h1>
        <p class="results__sub">{{ t('results.emptySub') }}</p>
        <p v-if="near.length" class="results__sub results__sub--dim">
          {{ t('results.emptyNear') }}
        </p>
      </div>

      <template v-else-if="single">
        <div class="results__head">
          <div class="label label--accent results__kicker">{{ t('results.winnerKicker') }}</div>
          <h1 class="results__title">{{ t('results.winnerTitle') }}</h1>
        </div>
        <button type="button" class="results__hero" @click="open(matches[0]!)">
          <img
            v-if="matches[0]!.movie.posterPath"
            class="results__hero-img"
            :src="imageUrl(matches[0]!.movie.posterPath, POSTER_FULL)"
            :alt="matches[0]!.movie.title"
          />
          <div class="results__hero-meta">
            <div class="clamp-2 results__hero-title">{{ matches[0]!.movie.title }}</div>
            <div class="results__hero-sub">
              {{ matches[0]!.movie.year
              }}<template v-if="matches[0]!.movie.rating"> · ★ {{ matches[0]!.movie.rating.toFixed(1) }}</template>
            </div>
          </div>
        </button>
        <div class="results__all-liked">
          <Icon name="heart" filled :size="13" />
          {{ t('results.allLiked', { count: state.participants.length }) }}
        </div>
      </template>

      <template v-else>
        <div class="results__head">
          <div class="label label--accent results__kicker">
            {{ t('results.manyKicker', { count: matches.length }) }}
          </div>
          <h1 class="results__title">{{ t('results.manyTitle') }}</h1>
          <p class="results__sub">{{ t('results.manySub') }}</p>
        </div>
        <div class="results__grid">
          <button
            v-for="m in matches"
            :key="m.movie.id"
            type="button"
            class="results__tile"
            @click="open(m)"
          >
            <img
              v-if="m.movie.posterPath"
              class="results__tile-img"
              :src="imageUrl(m.movie.posterPath, POSTER_THUMB)"
              :alt="m.movie.title"
            />
            <div class="results__tile-meta">
              <div class="clamp-2 results__tile-title">{{ m.movie.title }}</div>
              <div class="results__tile-sub">
                {{ m.movie.year }}<template v-if="m.movie.rating"> · ★ {{ m.movie.rating.toFixed(1) }}</template>
              </div>
            </div>
          </button>
        </div>
        <p class="results__tap-hint">{{ tapHint }}</p>
      </template>

      <template v-if="results?.stats.length">
        <div class="label results__label">{{ t('results.statsLabel') }}</div>
        <div class="results__stats">
          <div v-for="s in results!.stats" :key="s.id" class="results__stat">
            <span class="results__stat-avatar">
              {{ state.participants.find((p) => p.id === s.id)?.avatar || '🙂' }}
            </span>
            <div class="results__stat-num">{{ t('results.liked', { count: s.likes }) }}</div>
            <div class="results__stat-name">{{ s.name }}</div>
          </div>
        </div>
      </template>

      <template v-if="near.length">
        <div class="label results__label">{{ t('results.nearLabel') }}</div>
        <div class="results__near">
          <button
            v-for="n in near"
            :key="n.movie.id"
            type="button"
            class="results__near-row"
            @click="open(n)"
          >
            <img
              v-if="n.movie.posterPath"
              class="results__near-img"
              :src="imageUrl(n.movie.posterPath, POSTER_THUMB)"
              :alt="n.movie.title"
            />
            <div v-else class="results__near-img poster-fallback"></div>
            <div class="results__near-text">
              <div class="results__near-title">{{ n.movie.title }}</div>
              <div class="results__near-sub">{{ holdOut(n) }}</div>
            </div>
            <Icon name="chevron-right" :size="18" class="results__chev" />
          </button>
        </div>
      </template>
    </div>

    <div class="results__foot">
      <template v-if="single && canRequest">
        <QualityPicker v-if="!store.isRequested(matches[0]!.movie.id)" :disabled="sending" />
        <button
          class="btn btn--primary"
          type="button"
          :disabled="store.isRequested(matches[0]!.movie.id) || sending"
          @click="requestWinner"
        >
          <Spinner v-if="sending" />
          <template v-else-if="store.isRequested(matches[0]!.movie.id)">
            {{ t('detail.requested') }} <Icon name="check" :size="16" />
          </template>
          <template v-else>{{ t('detail.request') }} <Icon name="download" :size="16" /></template>
        </button>
      </template>
      <button class="btn btn--secondary" type="button" @click="store.leave()">
        {{ t('results.back') }}
      </button>
    </div>

    <DetailSheet v-if="opened" :movie="opened.movie" :votes="opened.votes" @close="opened = null" />

    <ConfirmSheet
      v-if="leaveOpen"
      :title="t('results.backTitle')"
      :text="t('results.backText')"
      :confirm-label="t('results.back')"
      :cancel-label="t('results.backCancel')"
      @confirm="confirmLeave"
      @close="leaveOpen = false"
    />
  </div>
</template>

<style>
.results {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.results__scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 0;
}

.results__hero-empty {
  text-align: center;
  padding: 32px 0 24px;
}

.results__head {
  text-align: center;
  margin-bottom: 18px;
}

.results__kicker {
  margin-bottom: 6px;
}

.results__title {
  font-size: var(--fs-5);
  font-weight: 600;
  margin: 0 0 4px;
}

.results__sub {
  font-size: var(--fs-2);
  color: var(--text-2);
  margin: 0 0 8px;
  line-height: 1.55;
}

.results__sub--dim {
  color: var(--text-3);
}

.results__hero {
  display: block;
  width: 150px;
  aspect-ratio: 2 / 3;
  border-radius: var(--radius-l);
  overflow: hidden;
  position: relative;
  background: var(--surface);
  border: 1px solid var(--border);
  margin: 0 auto 14px;
  padding: 0;
}

.results__hero-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.results__hero-meta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 12px;
  background: var(--scrim);
  text-align: left;
}

.results__hero-title {
  font-size: var(--fs-2);
  font-weight: 700;
}

.results__hero-sub {
  font-size: var(--fs-1);
  color: var(--text-on-image);
}

.results__all-liked {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: var(--fs-2);
  color: var(--online);
  margin-bottom: 22px;
}

.results__grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 10px;
}

.results__tile {
  position: relative;
  border-radius: var(--radius-s);
  overflow: hidden;
  aspect-ratio: 2 / 3;
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 0;
}

.results__tile-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.results__tile-meta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 8px;
  background: var(--scrim);
  text-align: left;
}

.results__tile-title {
  font-size: var(--fs-2);
  font-weight: 700;
  line-height: 1.15;
  margin-bottom: 2px;
}

.results__tile-sub {
  font-size: var(--fs-1);
  color: var(--text-on-image);
}

.results__tap-hint {
  text-align: center;
  font-size: var(--fs-1);
  color: var(--text-3);
  margin: 0 0 22px;
}

.results__label {
  margin-bottom: 10px;
}

.results__stats {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.results__stat {
  flex: 1;
  background: var(--surface);
  border-radius: var(--radius-m);
  padding: 12px;
  text-align: center;
}

.results__stat-avatar {
  font-size: 22px;
}

.results__stat-num {
  font-size: var(--fs-3);
  font-weight: 700;
  margin-top: 4px;
}

.results__stat-name {
  font-size: var(--fs-2);
  color: var(--text-body);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.results__near {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
}

.results__near-row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--surface);
  border: none;
  border-radius: var(--radius-m);
  padding: 9px 12px;
  text-align: left;
}

.results__near-img {
  width: 34px;
  height: 50px;
  border-radius: var(--radius-xs);
  object-fit: cover;
  flex: none;
}

.results__near-text {
  flex: 1;
  min-width: 0;
}

.results__near-title {
  font-size: var(--fs-2);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.results__near-sub {
  font-size: var(--fs-1);
  color: var(--text-2);
}

.results__chev {
  color: var(--muted);
}

.results__foot {
  flex: none;
  padding: 12px 24px 26px;
  border-top: 1px solid var(--border-soft);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

</style>
