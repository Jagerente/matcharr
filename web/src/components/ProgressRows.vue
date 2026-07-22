<script setup lang="ts">
import { computed } from 'vue'
import type { Participant } from '../api/client'
import { t } from '../i18n'
import * as store from '../store/session'

const props = defineProps<{ participants: Participant[]; departed?: Participant[]; deckSize: number }>()

const rows = computed(() =>
  props.participants.map((p) => ({
    ...p,
    you: p.id === store.you.value?.id,
    pct: props.deckSize ? Math.min(100, (p.progress / props.deckSize) * 100) : 0,
  })),
)
</script>

<template>
  <div class="progress-rows">
    <div v-for="p in rows" :key="p.id" class="progress-rows__row">
      <span class="progress-rows__avatar">{{ p.avatar || '🙂' }}</span>
      <span class="progress-rows__name" :class="{ 'progress-rows__name--you': p.you }">
        {{ p.you ? t('progress.you') : p.name }}
      </span>
      <div class="progress-rows__bar">
        <div
          class="progress-rows__fill"
          :class="{ 'progress-rows__fill--done': p.done }"
          :style="{ width: p.pct + '%' }"
        ></div>
      </div>
      <span class="progress-rows__count" :class="{ 'progress-rows__count--done': p.done }">
        {{ p.done ? t('progress.done') : `${p.progress}/${deckSize}` }}
      </span>
    </div>

    <div v-for="d in departed ?? []" :key="d.id" class="progress-rows__row progress-rows__row--left">
      <span class="progress-rows__avatar progress-rows__avatar--left">{{ d.avatar || '🙂' }}</span>
      <span class="progress-rows__name">{{ d.name }}</span>
      <div class="progress-rows__bar">
        <div
          class="progress-rows__fill progress-rows__fill--left"
          :style="{ width: (deckSize ? Math.min(100, (d.progress / deckSize) * 100) : 0) + '%' }"
        ></div>
      </div>
      <span class="progress-rows__count">{{ t('progress.left') }}</span>
    </div>
  </div>
</template>

<style>
.progress-rows {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.progress-rows__row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-rows__row--left {
  opacity: 0.5;
}

.progress-rows__avatar {
  font-size: 22px;
}

.progress-rows__avatar--left {
  filter: grayscale(1);
}

.progress-rows__name {
  font-size: var(--fs-2);
  width: 62px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-rows__name--you {
  font-weight: 600;
}

.progress-rows__bar {
  flex: 1;
  height: 6px;
  background: var(--raised);
  border-radius: var(--radius-pill);
  overflow: hidden;
}

.progress-rows__fill {
  height: 100%;
  background: var(--accent);
  border-radius: var(--radius-pill);
  transition: width var(--t-screen) var(--ease);
}

.progress-rows__fill--done {
  background: var(--online);
}

.progress-rows__fill--left {
  background: var(--raised-2);
}

.progress-rows__count {
  font-size: var(--fs-1);
  color: var(--text-2);
  width: 44px;
  text-align: right;
}

.progress-rows__count--done {
  color: var(--online);
}
</style>
