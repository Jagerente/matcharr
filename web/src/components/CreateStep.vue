<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchPresets, type Filters, type MatchMode, type Preset, type Settings } from '../api/client'
import { pickLocalized, t } from '../i18n'
import * as store from '../store/session'
import AppHeader from './AppHeader.vue'
import RangeSlider from './RangeSlider.vue'
import FiltersSheet from './FiltersSheet.vue'
import SkeletonChips from './SkeletonChips.vue'
import Spinner from './Spinner.vue'
import Icon from './Icon.vue'

defineProps<{ pending: boolean }>()
const emit = defineEmits<{ back: []; create: [settings: Settings] }>()

const presets = ref<Preset[]>([])
const presetsLoading = ref(true)
const presetsFailed = ref(false)
const presetId = ref('')
const matchMode = ref<MatchMode>('first')
const matchTarget = ref(3)
const deckLimit = ref(50)

const filters = ref<Filters | undefined>(undefined)
const threshold = ref(0)
const filtersOpen = ref(false)

const filtersCount = computed(() => {
  let n = filters.value ? Object.keys(filters.value).length : 0
  if (threshold.value) n++
  return n
})

async function loadPresets() {
  presetsLoading.value = true
  presetsFailed.value = false
  try {
    presets.value = await fetchPresets()
    presetId.value = presets.value[0]?.id ?? ''
  } catch {
    presetsFailed.value = true
  } finally {
    presetsLoading.value = false
  }
}

onMounted(() => {
  void store.genres().catch(() => {})
  void loadPresets()
})

function onSourcesWheel(e: WheelEvent) {
  ;(e.currentTarget as HTMLElement).scrollLeft += e.deltaY + e.deltaX
}

function bumpTarget(delta: number) {
  matchTarget.value = Math.min(20, Math.max(1, matchTarget.value + delta))
}

function onFiltersApply(next: Filters | undefined, nextThreshold: number) {
  filters.value = next
  threshold.value = nextThreshold
  filtersOpen.value = false
}

function submit() {
  emit('create', {
    presetId: presetId.value,
    matchMode: matchMode.value,
    matchTarget: matchMode.value === 'count' ? matchTarget.value : undefined,
    threshold: threshold.value || undefined,
    deckLimit: deckLimit.value,
    filters: filters.value,
  })
}
</script>

<template>
  <div class="create">
    <AppHeader>
      <template #left>
        <button class="btn-icon" type="button" @click="emit('back')">
          <Icon name="arrow-left" :size="18" />
        </button>
      </template>
    </AppHeader>
    <div class="create__body">
      <h1 class="create__title">{{ t('create.title') }}</h1>
      <p class="create__hint">{{ t('create.hint') }}</p>

      <div class="label create__label">{{ t('create.source') }}</div>
      <SkeletonChips v-if="presetsLoading" class="create__sources-skeleton" :height="40" />
      <button
        v-else-if="presetsFailed || !presets.length"
        type="button"
        class="create__sources-retry"
        @click="loadPresets"
      >
        <Icon name="undo" :size="15" />
        {{ t('create.sourcesRetry') }}
      </button>
      <div v-else class="create__sources" @wheel.prevent="onSourcesWheel">
        <button
          v-for="p in presets"
          :key="p.id"
          type="button"
          class="create__source"
          :class="{ 'create__source--active': p.id === presetId }"
          @click="presetId = p.id"
        >
          {{ p.icon }} {{ pickLocalized(p.title) }}
        </button>
      </div>

      <div class="label create__label">{{ t('create.mode') }}</div>
      <div class="create__modes">
        <button
          type="button"
          class="create__mode"
          :class="{ 'create__mode--active': matchMode === 'first' }"
          @click="matchMode = 'first'"
        >
          <span class="create__radio" :class="{ 'create__radio--on': matchMode === 'first' }"></span>
          <span class="create__mode-text">
            <span class="create__mode-name">{{ t('create.modeFirst') }}</span>
            <span class="create__mode-desc">{{ t('create.modeFirstDesc') }}</span>
          </span>
        </button>

        <button
          type="button"
          class="create__mode"
          :class="{ 'create__mode--active': matchMode === 'count' }"
          @click="matchMode = 'count'"
        >
          <span class="create__radio" :class="{ 'create__radio--on': matchMode === 'count' }"></span>
          <span class="create__mode-text">
            <span class="create__mode-name">{{ t('create.modeCount') }}</span>
            <span class="create__mode-desc">{{ t('create.modeCountDesc') }}</span>
          </span>
          <span class="create__stepper" @click.stop>
            <button type="button" class="create__step" @click="bumpTarget(-1)">
              <Icon name="minus" :size="13" />
            </button>
            <span class="create__step-value">{{ matchTarget }}</span>
            <button type="button" class="create__step" @click="bumpTarget(1)">
              <Icon name="plus" :size="13" />
            </button>
          </span>
        </button>

        <button
          type="button"
          class="create__mode"
          :class="{ 'create__mode--active': matchMode === 'all' }"
          @click="matchMode = 'all'"
        >
          <span class="create__radio" :class="{ 'create__radio--on': matchMode === 'all' }"></span>
          <span class="create__mode-text">
            <span class="create__mode-name">{{ t('create.modeAll') }}</span>
            <span class="create__mode-desc">{{ t('create.modeAllDesc') }}</span>
          </span>
        </button>
      </div>

      <div class="create__deck-row">
        <span class="label">{{ t('create.deckSize') }}</span>
        <span class="create__deck-value">{{ t('create.deckValue', { count: deckLimit }) }}</span>
      </div>
      <RangeSlider v-model="deckLimit" class="create__slider" :min="25" :max="100" :step="5" />

      <button type="button" class="create__filters" @click="filtersOpen = true">
        {{ t('create.filters') }}
        <span v-if="filtersCount" class="create__filters-badge">{{ filtersCount }}</span>
        <Icon name="chevron-down" :size="14" class="create__filters-chev" />
      </button>
    </div>

    <div class="create__foot">
      <button class="btn btn--primary" type="button" :disabled="pending || !presetId" @click="submit">
        <Spinner v-if="pending" />
        <template v-else>{{ t('create.cta') }}</template>
      </button>
    </div>

    <FiltersSheet
      v-if="filtersOpen"
      :filters="filters"
      :threshold="threshold"
      @close="filtersOpen = false"
      @apply="onFiltersApply"
    />
  </div>
</template>

<style>
.create {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.create__body {
  flex: 1;
  overflow-y: auto;
  padding: 10px 22px 8px;
}

.create__title {
  font-size: var(--fs-5);
  font-weight: 600;
  margin: 0 0 2px;
}

.create__hint {
  font-size: var(--fs-2);
  color: var(--text-2);
  margin: 0 0 16px;
}

.create__label {
  margin-bottom: 10px;
}

.create__sources {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: none;
  margin-bottom: 22px;
  padding-bottom: 2px;
}

.create__sources::-webkit-scrollbar {
  display: none;
}

/* the placeholder row keeps the chip row's exact footprint, so the real
   sources land without shifting anything below them */
.create__sources-skeleton,
.create__sources-retry {
  margin-bottom: 24px;
}

.create__sources-retry {
  height: 40px;
  padding: 0 16px;
  border-radius: var(--radius-pill);
  background: var(--surface);
  border: 1px solid var(--accent-line);
  color: var(--accent-text);
  font-size: var(--fs-2);
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.create__source {
  flex: none;
  padding: 9px 14px;
  border-radius: var(--radius-pill);
  background: var(--surface);
  border: 1px solid transparent;
  color: var(--text-2);
  font-size: var(--fs-2);
  white-space: nowrap;
}

.create__source--active {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--on-accent);
  font-weight: 600;
}

.create__modes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 22px;
}

.create__mode {
  border-radius: var(--radius-m);
  background: var(--surface);
  border: 1px solid transparent;
  padding: 12px 14px;
  display: flex;
  gap: 12px;
  align-items: center;
  text-align: left;
}

.create__mode--active {
  background: var(--raised);
  border-color: var(--accent-border-strong);
}

/* :active bubbles up from the nested stepper buttons */
.create__mode:active {
  transform: none;
}

.create__radio {
  width: 18px;
  height: 18px;
  border-radius: var(--radius-pill);
  border: 2px solid var(--muted);
  flex: none;
}

.create__radio--on {
  border: 5px solid var(--accent);
  background: var(--on-accent);
}

.create__mode-text {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.create__mode-name {
  font-size: var(--fs-3);
  font-weight: 600;
}

.create__mode-desc {
  font-size: var(--fs-2);
  color: var(--text-3);
  line-height: 1.35;
  margin-top: 2px;
}

.create__stepper {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--raised);
  border-radius: var(--radius-s);
  padding: 4px;
}

.create__step {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-xs);
  background: var(--bg);
  border: none;
  color: var(--text-2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.create__step-value {
  font-size: var(--fs-2);
  font-weight: 600;
  min-width: 14px;
  text-align: center;
}

.create__deck-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
}

.create__deck-value {
  font-size: var(--fs-3);
  font-weight: 600;
  color: var(--text);
}

.create__slider {
  margin: 0 0 16px;
}

.create__filters {
  width: 100%;
  height: 46px;
  background: var(--surface);
  border: 1px solid transparent;
  border-radius: var(--radius-m);
  color: var(--text-2);
  font-weight: 600;
  font-size: var(--fs-2);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.create__filters-badge {
  background: var(--accent);
  color: var(--on-accent);
  border-radius: var(--radius-pill);
  padding: 0 8px;
  font-size: var(--fs-1);
}

.create__filters-chev {
  color: var(--muted);
}

.create__foot {
  flex: none;
  padding: 14px 22px 26px;
}
</style>
