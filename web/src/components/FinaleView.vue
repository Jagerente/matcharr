<script setup lang="ts">
import { computed, ref } from 'vue'
import { imageUrl, POSTER_FULL, POSTER_THUMB } from '../api/client'
import * as store from '../store/session'
import { useBackLayer } from '../nav'
import { t } from '../i18n'
import QualityPicker from './QualityPicker.vue'
import StatTiles from './StatTiles.vue'
import Spinner from './Spinner.vue'
import Icon from './Icon.vue'

const state = computed(() => store.state.value!)
const kind = computed(() => store.finale.value)
const winner = computed(() => state.value.matches[0] ?? null)
const canSend = computed(() => store.canRequest.value && winner.value !== null)
const requested = computed(() => !!winner.value && store.isRequested(winner.value.movie.id))

useBackLayer(() => store.dismissFinale())

const sending = ref(false)

const deckStats = computed(() => [
  {
    value: state.value.matches.length,
    label: t('stats.matches', { count: state.value.matches.length }),
    accent: true,
  },
  { value: state.value.nearMissCount, label: t('stats.nearMisses') },
])

async function sendToServer() {
  if (!winner.value) return
  sending.value = true
  try {
    if (await store.requestMovie(winner.value.movie.id)) {
      setTimeout(() => store.dismissFinale(), 900)
    }
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="finale" :class="{ 'finale--rose': kind !== 'deck' }">
    <template v-if="kind === 'match' && winner">
      <div class="finale__body">
        <div class="label label--accent label--loud finale__kicker">
          {{ t('finale.matchKicker') }}
        </div>
        <div class="finale__poster-wrap">
          <img
            v-if="winner.movie.posterPath"
            class="finale__poster"
            :src="imageUrl(winner.movie.posterPath, POSTER_FULL)"
            :alt="winner.movie.title"
          />
          <div class="finale__poster-meta">
            <div class="finale__poster-title">{{ winner.movie.title }}</div>
            <div class="finale__poster-sub">
              {{ winner.movie.year }}<template v-if="winner.movie.rating"> · ★ {{ winner.movie.rating.toFixed(1) }}</template>
            </div>
          </div>
        </div>
        <h1 class="finale__title">{{ t('finale.matchTitle') }}</h1>
        <div class="finale__faces">
          <span v-for="p in state.participants" :key="p.id">{{ p.avatar || '🙂' }}</span>
        </div>
        <p class="finale__sub">{{ t('finale.matchSub', { count: state.participants.length }) }}</p>
      </div>
      <div class="finale__foot">
        <QualityPicker v-if="canSend && !requested" :disabled="sending" />
        <button v-if="canSend" class="btn btn--primary" type="button" :disabled="requested || sending" @click="sendToServer">
          <Spinner v-if="sending" />
          <template v-else-if="requested">
            {{ t('detail.requested') }} <Icon name="check" :size="16" />
          </template>
          <template v-else>{{ t('finale.send') }} <Icon name="download" :size="16" /></template>
        </button>
        <button
          :class="canSend ? 'btn btn--secondary' : 'btn btn--primary'"
          type="button"
          @click="store.dismissFinale()"
        >
          {{ t('finale.seeResults') }} <Icon name="arrow-right" :size="16" />
        </button>
      </div>
    </template>

    <template v-else-if="kind === 'goal'">
      <div class="finale__body">
        <div class="label label--accent label--loud finale__kicker">
          {{ t('finale.goalKicker') }}
        </div>
        <h1 class="finale__title">
          {{ t('finale.goalTitle', { count: state.matches.length }) }}
        </h1>
        <p class="finale__sub finale__sub--gap">{{ t('finale.goalSub') }}</p>
        <div class="finale__tiles">
          <div v-for="m in state.matches.slice(0, 3)" :key="m.movie.id" class="finale__tile">
            <img
              v-if="m.movie.posterPath"
              class="finale__tile-img"
              :src="imageUrl(m.movie.posterPath, POSTER_THUMB)"
              :alt="m.movie.title"
            />
            <div class="finale__tile-meta">
              <div class="clamp-2 finale__tile-title">{{ m.movie.title }}</div>
              <div class="finale__tile-sub">
                {{ m.movie.year }}<template v-if="m.movie.rating"> · ★ {{ m.movie.rating.toFixed(1) }}</template>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="finale__foot">
        <button class="btn btn--primary" type="button" @click="store.dismissFinale()">
          {{ t('finale.seeResults') }} <Icon name="arrow-right" :size="16" />
        </button>
      </div>
    </template>

    <template v-else>
      <div class="finale__body">
        <div class="hero-glyph">🏁</div>
        <div class="label label--accent label--loud finale__kicker">
          {{ t('finale.deckKicker') }}
        </div>
        <h1 class="finale__title">{{ t('finale.deckTitle', { count: state.deckSize }) }}</h1>
        <p class="finale__sub finale__sub--gap">{{ t('finale.deckSub') }}</p>
        <StatTiles :stats="deckStats" />
      </div>
      <div class="finale__foot">
        <button class="btn btn--primary" type="button" @click="store.dismissFinale()">
          {{ t('finale.seeResults') }} <Icon name="arrow-right" :size="16" />
        </button>
      </div>
    </template>
  </div>
</template>

<style>
.finale {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.finale--rose {
  background: radial-gradient(120% 80% at 50% 30%, #2a0a18, #0d0810 70%);
}

.finale__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0 28px;
  text-align: center;
}

.finale__kicker {
  margin-bottom: 18px;
}

.finale__poster-wrap {
  width: 184px;
  aspect-ratio: 2 / 3;
  border-radius: var(--radius-l);
  overflow: hidden;
  position: relative;
  background: var(--surface);
  box-shadow: var(--glow-match), var(--shadow-card);
  margin-bottom: 24px;
  animation: match-pop 0.45s var(--ease);
}

.finale__poster {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.finale__poster-meta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 14px;
  background: var(--scrim);
  text-align: left;
}

.finale__poster-title {
  font-size: var(--fs-3);
  font-weight: 700;
}

.finale__poster-sub {
  font-size: var(--fs-1);
  color: var(--text-on-image);
}

.finale__title {
  font-size: var(--fs-6);
  font-weight: 600;
  margin: 0 0 10px;
  line-height: 1.2;
}

.finale__faces {
  display: flex;
  margin-bottom: 6px;
}

.finale__faces span {
  font-size: 26px;
}

.finale__faces span + span {
  margin-left: -6px;
}

.finale__sub {
  font-size: var(--fs-2);
  color: var(--text-2);
  margin: 0;
}

.finale__sub--gap {
  margin-bottom: 26px;
}

.finale__tiles {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.finale__tile {
  width: 88px;
  aspect-ratio: 2 / 3;
  border-radius: var(--radius-m);
  overflow: hidden;
  position: relative;
  background: var(--surface);
  border: 1px solid var(--border);
}

.finale__tile-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.finale__tile-meta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 7px;
  background: var(--scrim);
  text-align: left;
}

.finale__tile-title {
  font-size: var(--fs-1);
  font-weight: 700;
  line-height: 1.1;
  margin-bottom: 2px;
}

.finale__tile-sub {
  font-size: var(--fs-1);
  color: var(--text-on-image);
}

.finale__foot {
  flex: none;
  padding: 0 28px 30px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
