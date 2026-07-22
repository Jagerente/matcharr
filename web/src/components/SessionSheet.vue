<script setup lang="ts">
import { computed, ref } from 'vue'
import { imageUrl, POSTER_THUMB } from '../api/client'
import type { Match } from '../api/client'
import { t } from '../i18n'
import * as store from '../store/session'
import ProgressRows from './ProgressRows.vue'
import DetailSheet from './DetailSheet.vue'
import Sheet from './Sheet.vue'
import Icon from './Icon.vue'

const props = defineProps<{ tab: 'matches' | 'progress' }>()
const emit = defineEmits<{ close: [] }>()

const active = ref(props.tab)
const state = computed(() => store.state.value!)
const opened = ref<Match | null>(null)

const goalLine = computed(() => {
  const s = state.value.settings
  if (s.matchMode === 'count') {
    return t('sessionSheet.goalLine', { target: s.matchTarget ?? 0 })
  }
  return t('sessionSheet.tapLine')
})
</script>

<template>
  <div>
    <Sheet class="session-sheet" @close="emit('close')">
      <div class="session-sheet__tabs">
        <button
          type="button"
          class="session-sheet__tab"
          :class="{ 'session-sheet__tab--on': active === 'matches' }"
          @click="active = 'matches'"
        >
          <Icon name="heart" filled :size="13" />
          {{ t('sessionSheet.matchesTab', { count: state.matches.length }) }}
        </button>
        <button
          type="button"
          class="session-sheet__tab"
          :class="{ 'session-sheet__tab--on': active === 'progress' }"
          @click="active = 'progress'"
        >
          {{ t('sessionSheet.progressTab') }}
        </button>
      </div>

      <div class="session-sheet__body">
        <template v-if="active === 'matches'">
          <p v-if="!state.matches.length" class="session-sheet__empty">
            {{ t('sessionSheet.empty') }}
          </p>
          <template v-else>
            <button
              v-for="m in state.matches"
              :key="m.movie.id"
              type="button"
              class="session-sheet__row"
              @click="opened = m"
            >
              <img
                v-if="m.movie.posterPath"
                class="session-sheet__poster"
                :src="imageUrl(m.movie.posterPath, POSTER_THUMB)"
                :alt="m.movie.title"
              />
              <div v-else class="session-sheet__poster poster-fallback"></div>
              <span class="session-sheet__row-text">
                <span class="session-sheet__row-title">{{ m.movie.title }}</span>
                <span class="session-sheet__row-sub">
                  <Icon name="heart" filled :size="11" /> {{ t('sessionSheet.everyoneLiked') }}
                </span>
              </span>
              <Icon name="chevron-right" :size="18" class="session-sheet__chev" />
            </button>
            <p class="session-sheet__goal">{{ goalLine }}</p>
          </template>
        </template>

        <ProgressRows
          v-else
          :participants="state.participants"
          :departed="state.departed"
          :deck-size="state.deckSize"
        />
      </div>

      <button class="btn btn--secondary session-sheet__back" type="button" @click="emit('close')">
        {{ t('sessionSheet.back') }}
      </button>
    </Sheet>

    <DetailSheet v-if="opened" :movie="opened.movie" :votes="opened.votes" @close="opened = null" />
  </div>
</template>

<style>
.session-sheet {
  display: flex;
  flex-direction: column;
  max-height: 84dvh;
  min-height: 46dvh;
}

.session-sheet__tabs {
  display: flex;
  background: var(--bg);
  border-radius: var(--radius-m);
  padding: 4px;
  gap: 4px;
  margin-bottom: 18px;
  flex: none;
}

.session-sheet__tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 0;
  border-radius: var(--radius-s);
  border: none;
  background: none;
  font-size: var(--fs-2);
  color: var(--text-2);
}

.session-sheet__tab--on {
  background: var(--raised);
  color: var(--text);
  font-weight: 600;
}

.session-sheet__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.session-sheet__empty {
  color: var(--text-3);
  font-size: var(--fs-2);
  text-align: center;
  margin: 24px 0;
}

.session-sheet__row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--raised);
  border: none;
  border-radius: var(--radius-m);
  padding: 10px 12px;
  text-align: left;
}

.session-sheet__poster {
  width: 40px;
  height: 60px;
  border-radius: var(--radius-xs);
  object-fit: cover;
  flex: none;
}

.session-sheet__row-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.session-sheet__row-title {
  font-size: var(--fs-3);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-sheet__row-sub {
  font-size: var(--fs-1);
  color: var(--online);
  display: flex;
  align-items: center;
  gap: 5px;
}

.session-sheet__chev {
  color: var(--muted);
}

.session-sheet__goal {
  text-align: center;
  font-size: var(--fs-2);
  color: var(--text-3);
  margin: 6px 0 0;
}

.session-sheet__back {
  margin-top: 14px;
  flex: none;
}
</style>
