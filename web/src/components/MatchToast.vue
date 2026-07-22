<script setup lang="ts">
import { computed } from 'vue'
import { imageUrl, POSTER_THUMB } from '../api/client'
import * as store from '../store/session'
import { useBackLayer } from '../nav'
import { t } from '../i18n'

const count = computed(() => store.state.value?.matches.length ?? 0)

useBackLayer(
  () => {
    store.matchToast.value = null
  },
  () => store.matchToast.value !== null,
)
</script>

<template>
  <Transition name="toast">
    <div
      v-if="store.matchToast.value"
      class="match-toast-wrap"
      @click="store.matchToast.value = null"
    >
      <div class="match-toast">
        <div class="label label--accent label--loud match-toast__kicker">
          {{ t('matchToast.kicker', { count }) }}
        </div>
        <img
          v-if="store.matchToast.value.posterPath"
          class="match-toast__poster"
          :src="imageUrl(store.matchToast.value.posterPath, POSTER_THUMB)"
          :alt="store.matchToast.value.title"
        />
        <div class="match-toast__title">{{ store.matchToast.value.title }}</div>
        <div class="match-toast__sub">{{ t('matchToast.sub') }}</div>
      </div>
    </div>
  </Transition>
</template>

<style>
.match-toast-wrap {
  position: fixed;
  inset: 0;
  z-index: 80;
  background: var(--backdrop);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  cursor: pointer;
}

.match-toast {
  width: 100%;
  max-width: 340px;
  background: var(--surface);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-xl);
  padding: 22px;
  text-align: center;
  box-shadow: var(--glow-match);
  animation: match-pop 0.45s var(--ease);
}

.match-toast__kicker {
  margin-bottom: 10px;
}

.match-toast__poster {
  width: 90px;
  border-radius: var(--radius-m);
  border: 1px solid var(--border);
  margin: 0 auto 14px;
}

.match-toast__title {
  font-size: var(--fs-4);
  font-weight: 700;
  margin-bottom: 4px;
}

.match-toast__sub {
  font-size: var(--fs-2);
  color: var(--text-2);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity var(--t-screen);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
}
</style>
