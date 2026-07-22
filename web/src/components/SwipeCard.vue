<script setup lang="ts">
import { computed, ref } from 'vue'
import { imageUrl, POSTER_FULL, type Movie } from '../api/client'
import { t } from '../i18n'
import * as store from '../store/session'
import Icon from './Icon.vue'

const props = defineProps<{ movie: Movie }>()
const emit = defineEmits<{ verdict: [v: 'like' | 'nope'] }>()

const dx = ref(0)
const dy = ref(0)
const dragging = ref(false)
// A tap wiggles a pixel or two; the card must not follow until movement
// clears the dead zone or the poster twitches on every tap.
const moved = ref(false)
const leaving = ref<'like' | 'nope' | null>(null)
const details = ref(false)
const posterReady = ref(false)
// Tap-through guard: the tap that opens the overlay synthesizes a click
// hit-tested against the new overlay, which must not close it.
const overlayPressed = ref(false)
const full = ref<Movie | null>(null)

let startX = 0
let startY = 0

const SWIPE_THRESHOLD = 90
const DEAD_ZONE = 8

const shown = computed(() => full.value ?? props.movie)

const cardStyle = computed(() => {
  if (leaving.value) {
    const dir = leaving.value === 'like' ? 1 : -1
    return {
      transform: `translate(${dir * 120}vw, ${dy.value}px) rotate(${dir * 30}deg)`,
      transition: 'transform 0.35s ease-in',
    }
  }
  if (dragging.value && moved.value) {
    return {
      transform: `translate(${dx.value}px, ${dy.value}px) rotate(${dx.value / 18}deg)`,
    }
  }
  return { transform: 'translate(0, 0)', transition: 'transform 0.25s ease-out' }
})

const hint = computed(() => {
  if (Math.abs(dx.value) < 30) return null
  return dx.value > 0 ? 'like' : 'nope'
})

function onDown(e: PointerEvent) {
  if (leaving.value || details.value) return
  dragging.value = true
  moved.value = false
  startX = e.clientX
  startY = e.clientY
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
}

function onMove(e: PointerEvent) {
  if (!dragging.value) return
  dx.value = e.clientX - startX
  dy.value = e.clientY - startY
  if (Math.abs(dx.value) > DEAD_ZONE || Math.abs(dy.value) > DEAD_ZONE) {
    moved.value = true
  }
}

function onUp() {
  if (!dragging.value) return
  dragging.value = false
  if (moved.value && Math.abs(dx.value) > SWIPE_THRESHOLD) {
    fly(dx.value > 0 ? 'like' : 'nope')
    return
  }
  if (!moved.value) openDetails()
  dx.value = 0
  dy.value = 0
}

async function openDetails() {
  details.value = true
  overlayPressed.value = false
  full.value = (await store.movieDetails(props.movie.id)) ?? null
}

function onOverlayClick() {
  if (!overlayPressed.value) return
  overlayPressed.value = false
  details.value = false
}

function fly(verdict: 'like' | 'nope') {
  details.value = false
  leaving.value = verdict
  setTimeout(() => emit('verdict', verdict), 250)
}

defineExpose({ fly })
</script>

<template>
  <div class="swipe-card-stage">
    <div class="swipe-card-stage__behind"></div>
    <article
      class="swipe-card"
      :style="cardStyle"
      @pointerdown="onDown"
      @pointermove="onMove"
      @pointerup="onUp"
      @pointercancel="onUp"
      @contextmenu.prevent
    >
      <div
        v-if="movie.posterPath && !posterReady"
        class="swipe-card__poster skeleton"
        aria-hidden="true"
      ></div>
      <img
        v-if="movie.posterPath"
        class="swipe-card__poster"
        :class="{ 'swipe-card__poster--dim': details }"
        :src="imageUrl(movie.posterPath, POSTER_FULL)"
        :alt="movie.title"
        draggable="false"
        @load="posterReady = true"
        @error="posterReady = true"
      />
      <div
        v-else
        class="swipe-card__poster swipe-card__poster--empty poster-fallback"
        :class="{ 'swipe-card__poster--dim': details }"
      >
        <span class="swipe-card__no-poster">{{ t('swipe.noPoster') }}</span>
      </div>

      <span v-if="hint === 'like'" class="swipe-card__tag swipe-card__tag--like">
        {{ t('swipe.like') }}
      </span>
      <span v-if="hint === 'nope'" class="swipe-card__tag swipe-card__tag--nope">
        {{ t('swipe.pass') }}
      </span>

      <template v-if="!details">
        <div class="swipe-card__scrim"></div>
        <div class="swipe-card__meta">
          <h2 class="swipe-card__title">{{ movie.title }}</h2>
          <div class="swipe-card__facts">
            <span v-if="movie.year">{{ movie.year }}</span>
            <span v-if="movie.rating" class="swipe-card__rating">★ {{ movie.rating.toFixed(1) }}</span>
            <span v-if="movie.genres?.length">{{ movie.genres.slice(0, 2).join(' · ') }}</span>
          </div>
          <div class="swipe-card__tap-hint">
            <Icon name="info" :size="14" />{{ t('swipe.tapOverview') }}
          </div>
        </div>
      </template>

      <div
        v-else
        class="swipe-card__details"
        @pointerdown.stop="overlayPressed = true"
        @click="onOverlayClick"
      >
        <div class="swipe-card__details-top">
          <span class="sheet__handle swipe-card__details-handle"></span>
          <button type="button" class="swipe-card__close" @click.stop="details = false">
            <Icon name="x" :size="15" />
          </button>
        </div>
        <img
          v-if="movie.posterPath"
          class="swipe-card__details-poster"
          :src="imageUrl(movie.posterPath, POSTER_FULL)"
          :alt="movie.title"
        />
        <h2 class="swipe-card__details-title">{{ shown.title }}</h2>
        <div
          v-if="shown.originalTitle && shown.originalTitle !== shown.title"
          class="swipe-card__details-orig"
        >
          {{ shown.originalTitle }}
        </div>
        <div class="swipe-card__facts swipe-card__details-facts">
          <span v-if="shown.year">{{ shown.year }}</span>
          <span v-if="shown.rating" class="swipe-card__rating">★ {{ shown.rating.toFixed(1) }}</span>
          <span v-if="shown.runtime">{{ t('movie.runtime', { count: shown.runtime }) }}</span>
        </div>
        <p class="swipe-card__overview">{{ shown.overview || t('movie.noOverview') }}</p>
        <div class="swipe-card__chips">
          <span v-for="g in shown.genres ?? []" :key="g" class="swipe-card__chip">{{ g }}</span>
        </div>
        <div class="swipe-card__details-hint">
          <Icon name="chevron-down" :size="13" /> {{ t('swipe.tapClose') }}
        </div>
      </div>
    </article>
  </div>
</template>

<style>
.swipe-card-stage {
  position: relative;
  height: 100%;
}

.swipe-card-stage__behind {
  position: absolute;
  inset: 8px 10px 14px;
  background: var(--surface);
  border-radius: var(--radius-xl);
  transform: scale(0.96);
  opacity: 0.5;
}

.swipe-card {
  position: relative;
  height: 100%;
  border-radius: var(--radius-xl);
  overflow: hidden;
  background: var(--surface);
  box-shadow: var(--shadow-card);
  touch-action: none;
  user-select: none;
  -webkit-touch-callout: none;
  cursor: grab;
}

.swipe-card:active {
  cursor: grabbing;
}

.swipe-card__poster {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.swipe-card__poster--dim {
  filter: brightness(0.25);
}

.swipe-card__poster--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-3);
}

.swipe-card__no-poster {
  font-size: var(--fs-1);
}

.swipe-card__tag {
  position: absolute;
  top: 24px;
  padding: 6px 16px;
  border-radius: var(--radius-s);
  font-size: var(--fs-4);
  font-weight: 700;
  border: 3px solid;
  background: rgba(13, 12, 15, 0.6);
  z-index: 2;
}

.swipe-card__tag--like {
  left: 20px;
  color: var(--accent);
  border-color: var(--accent);
  transform: rotate(-8deg);
  box-shadow: var(--glow-hint);
}

.swipe-card__tag--nope {
  right: 20px;
  color: var(--text-2);
  border-color: var(--text-2);
  transform: rotate(8deg);
}

.swipe-card__scrim {
  position: absolute;
  inset: 0;
  background: var(--scrim-tall);
}

.swipe-card__meta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 22px 20px;
}

.swipe-card__title {
  font-size: var(--fs-6);
  font-weight: 700;
  margin: 0 0 8px;
  line-height: 1.1;
}

.swipe-card__facts {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: var(--fs-2);
  color: var(--text-on-image);
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.swipe-card__rating {
  color: var(--star);
}

.swipe-card__tap-hint {
  display: flex;
  align-items: center;
  gap: 7px;
  color: rgba(255, 255, 255, 0.62);
  font-size: var(--fs-2);
}

.swipe-card__details {
  position: absolute;
  inset: 0;
  padding: 20px 22px 24px;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  scrollbar-width: none;
  cursor: default;
}

.swipe-card__details::-webkit-scrollbar {
  display: none;
}

.swipe-card__details > * {
  animation: fade-rise var(--t-screen) var(--ease);
}

.swipe-card__details-top {
  position: relative;
  margin-bottom: 16px;
}

.swipe-card__details-handle {
  background: rgba(255, 255, 255, 0.24);
  margin-bottom: 0;
}

.swipe-card__close {
  position: absolute;
  right: -4px;
  top: -8px;
  width: 34px;
  height: 34px;
  border-radius: var(--radius-pill);
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: var(--text);
}

.swipe-card__details-poster {
  width: 76px;
  border-radius: var(--radius-s);
  border: 1px solid var(--border);
  margin-bottom: 16px;
}

.swipe-card__details-title {
  font-size: var(--fs-5);
  font-weight: 700;
  margin: 0 0 3px;
}

.swipe-card__details-orig {
  font-size: var(--fs-2);
  color: var(--text-3);
  font-style: italic;
  margin-bottom: 6px;
}

.swipe-card__details-facts {
  font-size: var(--fs-2);
  margin-bottom: 16px;
}

.swipe-card__overview {
  font-size: var(--fs-3);
  line-height: 1.6;
  color: var(--text-body);
  margin: 0 0 16px;
}

.swipe-card__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: auto;
}

.swipe-card__chip {
  padding: 6px 12px;
  border-radius: var(--radius-pill);
  background: rgba(255, 255, 255, 0.08);
  font-size: var(--fs-1);
  color: var(--text-on-image);
}

.swipe-card__details-hint {
  margin-top: 16px;
  font-size: var(--fs-1);
  color: var(--text-3);
  display: flex;
  align-items: center;
  gap: 6px;
}
</style>
