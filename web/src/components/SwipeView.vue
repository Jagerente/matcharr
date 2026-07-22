<script setup lang="ts">
import { computed, ref } from 'vue'
import * as store from '../store/session'
import { useBackGuard } from '../nav'
import { t } from '../i18n'
import SwipeCard from './SwipeCard.vue'
import SessionSheet from './SessionSheet.vue'
import ExitSheet from './ExitSheet.vue'
import ProgressRows from './ProgressRows.vue'
import AppHeader from './AppHeader.vue'
import PulseDots from './PulseDots.vue'
import Icon from './Icon.vue'

const state = computed(() => store.state.value!)
const movie = computed(() => store.currentMovie.value)
const pending = computed(() => store.deckPending.value)
const card = ref<InstanceType<typeof SwipeCard> | null>(null)

const sheetTab = ref<'matches' | 'progress' | null>(null)
const exitOpen = ref(false)

useBackGuard(() => {
  exitOpen.value = true
})

const others = computed(() => state.value.participants.slice(0, 3))

function verdictFromButton(v: 'like' | 'nope') {
  card.value?.fly(v)
}

function onVerdict(v: 'like' | 'nope') {
  void store.swipe(v)
}
</script>

<template>
  <div class="swipe">
    <template v-if="movie || pending">
      <AppHeader>
        <template #left>
          <button
            type="button"
            class="swipe__exit"
            :aria-label="store.isHost.value ? t('swipe.finish') : t('swipe.leave')"
            @click="exitOpen = true"
          >
            <Icon name="exit" :size="16" />
          </button>
        </template>
        <template #right>
          <button type="button" class="swipe__pill" @click="sheetTab = 'progress'">
            <span class="swipe__faces">
              <span v-for="p in others" :key="p.id">{{ p.avatar || '🙂' }}</span>
            </span>
            <span v-if="state.participants.length > 3" class="swipe__more">
              +{{ state.participants.length - 3 }}
            </span>
            <Icon name="chevron-down" :size="12" class="swipe__chev" />
          </button>
        </template>
      </AppHeader>

      <div class="swipe__progress">
        <div class="swipe__bar">
          <div
            class="swipe__fill"
            :style="{ width: (state.deckSize ? (store.cursor.value / state.deckSize) * 100 : 0) + '%' }"
          ></div>
        </div>
        <span class="swipe__count">{{ store.cursor.value }} / {{ state.deckSize }}</span>
      </div>

      <div class="swipe__stage">
        <SwipeCard v-if="movie" ref="card" :key="movie.id" :movie="movie" @verdict="onVerdict" />
        <div v-else class="swipe-card-stage">
          <div class="swipe-card-stage__behind"></div>
          <div class="swipe-card" aria-busy="true">
            <div class="skeleton swipe__dealing-sheen"></div>
            <div class="swipe__dealing">
              <PulseDots />
              <span>{{ t('swipe.dealing') }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="swipe__actions">
        <button
          type="button"
          class="swipe__btn swipe__btn--nope"
          :disabled="!movie"
          @click="verdictFromButton('nope')"
        >
          <Icon name="x" :size="26" />
        </button>
        <button
          type="button"
          class="swipe__btn swipe__btn--undo"
          :disabled="!movie || store.cursor.value === 0"
          @click="store.undo()"
        >
          <Icon name="undo" :size="20" />
        </button>
        <button
          type="button"
          class="swipe__btn swipe__btn--like"
          :disabled="!movie"
          @click="verdictFromButton('like')"
        >
          <Icon name="heart" filled :size="26" />
        </button>
      </div>
    </template>

    <template v-else>
      <AppHeader />
      <div class="swipe__done">
        <div class="swipe__done-hero">
          <div class="hero-glyph">🍿</div>
          <h1 class="swipe__done-title">{{ t('swipe.doneTitle') }}</h1>
          <p class="swipe__done-sub">{{ t('swipe.doneSub') }}</p>
        </div>
        <div class="label swipe__done-label">{{ t('swipe.progressLabel') }}</div>
        <ProgressRows
          :participants="state.participants"
          :departed="state.departed"
          :deck-size="state.deckSize"
        />
        <div v-if="state.matches.length" class="note swipe__done-banner">
          <Icon name="heart" filled :size="16" class="swipe__done-icon" />
          <span>{{ t('swipe.matchesSoFar', { count: state.matches.length }) }}</span>
        </div>
        <div class="swipe__done-foot">
          <button class="btn btn--secondary" type="button" @click="exitOpen = true">
            {{ store.isHost.value ? t('swipe.endNow') : t('swipe.leave') }}
          </button>
        </div>
      </div>
    </template>

    <SessionSheet v-if="sheetTab" :tab="sheetTab" @close="sheetTab = null" />
    <ExitSheet v-if="exitOpen" @close="exitOpen = false" />
  </div>
</template>

<style>
.swipe {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: clip;
}

.swipe__exit {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: var(--surface);
  border: 1px solid transparent;
  color: var(--text-2);
  border-radius: var(--radius-pill);
}

.swipe__pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--surface);
  border: 1px solid transparent;
  color: var(--text-2);
  border-radius: var(--radius-pill);
  padding: 6px 13px 6px 10px;
  font-weight: 600;
  font-size: var(--fs-2);
}

.swipe__chev {
  opacity: 0.7;
}

.swipe__faces {
  display: inline-flex;
}

.swipe__faces span {
  font-size: 15px;
}

.swipe__faces span + span {
  margin-left: -5px;
}

.swipe__more {
  font-size: var(--fs-1);
  font-weight: 600;
  color: var(--text-2);
}

.swipe__progress {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px 12px;
}

.swipe__bar {
  flex: 1;
  height: 5px;
  background: var(--raised);
  border-radius: var(--radius-pill);
  overflow: hidden;
}

.swipe__fill {
  height: 100%;
  background: var(--accent);
  border-radius: var(--radius-pill);
  transition: width var(--t-card) var(--ease);
}

.swipe__count {
  font-size: var(--fs-1);
  color: var(--text-2);
}

.swipe__stage {
  flex: 1;
  min-height: 0;
  padding: 0 24px;
}

.swipe__actions {
  flex: none;
  padding: 20px 24px 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
}

.swipe__btn {
  width: 60px;
  height: 60px;
  border-radius: var(--radius-pill);
  display: flex;
  align-items: center;
  justify-content: center;
}

.swipe__btn--nope {
  background: var(--surface);
  border: 1px solid transparent;
  color: var(--text-2);
}

.swipe__btn--undo {
  width: 52px;
  height: 52px;
  background: var(--surface-2);
  border: 1px solid transparent;
  color: var(--text-3);
}

.swipe__btn:disabled {
  opacity: 0.4;
}

.swipe__dealing-sheen {
  position: absolute;
  inset: 0;
}

.swipe__dealing {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  font-size: var(--fs-2);
  color: var(--text-2);
}

.swipe__btn--like {
  background: var(--accent);
  border: none;
  color: var(--on-accent);
}

.swipe__done {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding: 24px;
}

.swipe__done-hero {
  text-align: center;
  padding: 16px 0 28px;
}

.swipe__done-title {
  font-size: var(--fs-5);
  font-weight: 600;
  margin: 0 0 8px;
}

.swipe__done-sub {
  font-size: var(--fs-3);
  color: var(--text-2);
  margin: 0;
}

.swipe__done-label {
  margin-bottom: 12px;
}

.swipe__done-banner {
  margin-top: 28px;
}

.swipe__done-icon {
  color: var(--accent);
}

.swipe__done-foot {
  margin-top: auto;
  padding-top: 20px;
}
</style>
