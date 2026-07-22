<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { t } from './i18n'
import * as store from './store/session'
import EntryFlow from './components/EntryFlow.vue'
import LobbyView from './components/LobbyView.vue'
import SwipeView from './components/SwipeView.vue'
import ContinueView from './components/ContinueView.vue'
import ResultsView from './components/ResultsView.vue'
import FinaleView from './components/FinaleView.vue'
import EndNotice from './components/EndNotice.vue'
import MatchToast from './components/MatchToast.vue'
import ErrorToast from './components/ErrorToast.vue'

const loading = ref(true)

onMounted(async () => {
  await store.resume(store.invite.value || undefined)
  loading.value = false
})

const view = computed(() => {
  if (store.notice.value) return EndNotice
  const s = store.state.value
  if (!s) return EntryFlow
  switch (s.status) {
    case 'lobby':
      return LobbyView
    case 'voting':
      return ContinueView
    case 'finished':
      return store.finale.value ? FinaleView : ResultsView
    default:
      return s.allDone ? ContinueView : SwipeView
  }
})
</script>

<template>
  <div class="app">
    <span
      v-if="store.inSession.value && !store.connected.value"
      class="app__reconnect"
    >
      <span class="app__reconnect-dot"></span>{{ t('app.reconnecting') }}
    </span>

    <ErrorToast
      v-if="store.error.value"
      :key="store.error.value"
      :message="store.error.value"
      @close="store.error.value = ''"
    />

    <main class="app__main">
      <p v-if="loading" class="app__loading">{{ t('app.loading') }}</p>
      <component :is="view" v-else />
    </main>

    <MatchToast />
  </div>
</template>

<style>
.app {
  max-width: 480px;
  margin: 0 auto;
  height: 100dvh;
  background: var(--bg);
  display: flex;
  flex-direction: column;
  position: relative;
  /* clips dragged cards without touching the root scroller (pull-to-refresh) */
  overflow: clip;
}

@media (min-width: 520px) {
  .app {
    border-inline: 1px solid var(--border-soft);
  }
}

.app__reconnect {
  position: absolute;
  top: 64px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 90;
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--raised);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  padding: 6px 14px;
  font-size: var(--fs-2);
  color: var(--warn);
}

.app__reconnect-dot {
  width: 7px;
  height: 7px;
  border-radius: var(--radius-pill);
  background: var(--warn);
  animation: pulse 1s infinite;
}

.app__loading {
  padding: var(--space-6);
  color: var(--text-2);
}

.app__main {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  overflow-x: hidden;
}
</style>
