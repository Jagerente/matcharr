<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { Filters, Genre } from '../api/client'
import { t } from '../i18n'
import * as store from '../store/session'
import Sheet from './Sheet.vue'
import SkeletonChips from './SkeletonChips.vue'
import RangeSlider from './RangeSlider.vue'
import Icon from './Icon.vue'

const YEAR_MIN = 1950
const YEAR_MAX = 2026
const VOTE_STEPS = [0, 100, 500, 1000, 5000]

const props = defineProps<{ filters?: Filters; threshold: number }>()
const emit = defineEmits<{
  close: []
  apply: [filters: Filters | undefined, threshold: number]
}>()

const genres = ref<Genre[]>([])
const genresLoading = ref(true)
const selected = ref<Set<number>>(new Set(props.filters?.genreIds ?? []))
const yearFrom = ref(props.filters?.yearFrom ?? YEAR_MIN)
const yearTo = ref(props.filters?.yearTo ?? YEAR_MAX)
const ratingMin = ref(props.filters?.ratingMin ?? 0)
const votesIdx = ref(Math.max(0, VOTE_STEPS.indexOf(props.filters?.votesMin ?? 0)))
const anyN = ref(props.threshold > 0)
const thresholdN = ref(props.threshold || 3)

onMounted(async () => {
  try {
    genres.value = await store.genres()
  } catch {
    /* genre chips just don't render */
  } finally {
    genresLoading.value = false
  }
})

function toggle(id: number) {
  const next = new Set(selected.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selected.value = next
}

const yearLabel = computed(() => {
  const open = yearFrom.value === YEAR_MIN && yearTo.value === YEAR_MAX
  if (open) return t('filters.anyYear')
  const from = yearFrom.value > YEAR_MIN ? yearFrom.value : t('filters.anyYear')
  const to = yearTo.value < YEAR_MAX ? yearTo.value : t('filters.now')
  return `${from} – ${to}`
})

function bumpThreshold(delta: number) {
  thresholdN.value = Math.min(10, Math.max(2, thresholdN.value + delta))
}

function reset() {
  selected.value = new Set()
  yearFrom.value = YEAR_MIN
  yearTo.value = YEAR_MAX
  ratingMin.value = 0
  votesIdx.value = 0
  anyN.value = false
}

function apply() {
  const f: Filters = {}
  if (selected.value.size) f.genreIds = [...selected.value]
  if (yearFrom.value > YEAR_MIN) f.yearFrom = yearFrom.value
  if (yearTo.value < YEAR_MAX) f.yearTo = yearTo.value
  if (ratingMin.value > 0) f.ratingMin = ratingMin.value
  const votes = VOTE_STEPS[votesIdx.value] ?? 0
  if (votes > 0) f.votesMin = votes
  if (f.yearFrom && f.yearTo && f.yearFrom > f.yearTo) {
    ;[f.yearFrom, f.yearTo] = [f.yearTo, f.yearFrom]
  }
  emit('apply', Object.keys(f).length ? f : undefined, anyN.value ? thresholdN.value : 0)
}
</script>

<template>
  <Sheet class="filters" @close="emit('close')">
    <div class="filters__head">
        <span class="filters__title">{{ t('filters.title') }}</span>
        <span class="filters__optional">{{ t('filters.optional') }}</span>
      </div>
      <p class="filters__hint">{{ t('filters.hint') }}</p>

      <div class="filters__scroll">
        <div v-if="genresLoading" class="filters__group">
          <div class="label filters__label">{{ t('filters.genres') }}</div>
          <SkeletonChips wrap :count="8" />
        </div>
        <div v-else-if="genres.length" class="filters__group">
          <div class="label filters__label">{{ t('filters.genres') }}</div>
          <div class="filters__genres">
            <button
              v-for="g in genres"
              :key="g.id"
              type="button"
              class="filters__genre"
              :class="{ 'filters__genre--on': selected.has(g.id) }"
              @click="toggle(g.id)"
            >
              {{ g.name }}
            </button>
          </div>
        </div>

        <div class="filters__group">
          <div class="filters__row">
            <span class="label">{{ t('filters.year') }}</span>
            <span class="filters__value">{{ yearLabel }}</span>
          </div>
          <RangeSlider
            range
            v-model:from="yearFrom"
            v-model:to="yearTo"
            :min="YEAR_MIN"
            :max="YEAR_MAX"
          />
        </div>

        <div class="filters__group">
          <div class="filters__row">
            <span class="label">{{ t('filters.minRating') }}</span>
            <span class="filters__value">
              {{ ratingMin ? ratingMin.toFixed(1) + '+' : t('filters.any') }}
            </span>
          </div>
          <RangeSlider v-model="ratingMin" :min="0" :max="8" :step="0.5" />
        </div>

        <div class="filters__group">
          <div class="filters__row">
            <span class="label">{{ t('filters.minVotes') }}</span>
            <span class="filters__value">
              {{ VOTE_STEPS[votesIdx] ? VOTE_STEPS[votesIdx] + '+' : t('filters.any') }}
            </span>
          </div>
          <RangeSlider v-model="votesIdx" :min="0" :max="VOTE_STEPS.length - 1" />
        </div>

        <div class="filters__group">
          <div class="label filters__label">{{ t('filters.matchRule') }}</div>
          <div class="filters__segment">
            <button
              type="button"
              class="filters__segment-btn"
              :class="{ 'filters__segment-btn--on': !anyN }"
              @click="anyN = false"
            >
              {{ t('filters.everyone') }}
            </button>
            <button
              type="button"
              class="filters__segment-btn"
              :class="{ 'filters__segment-btn--on': anyN }"
              @click="anyN = true"
            >
              {{ t('filters.anyOfUsBefore') }}
              <span class="filters__mini-stepper" @click.stop>
                <button type="button" class="filters__mini-step" @click="bumpThreshold(-1)">
                  <Icon name="minus" :size="11" />
                </button>
                <b>{{ thresholdN }}</b>
                <button type="button" class="filters__mini-step" @click="bumpThreshold(1)">
                  <Icon name="plus" :size="11" />
                </button>
              </span>
              {{ t('filters.anyOfUsAfter') }}
            </button>
          </div>
        </div>
      </div>

    <div class="filters__foot">
      <button type="button" class="filters__reset" @click="reset">{{ t('filters.reset') }}</button>
      <button type="button" class="filters__apply" @click="apply">{{ t('filters.apply') }}</button>
    </div>
  </Sheet>
</template>

<style>
.filters {
  display: flex;
  flex-direction: column;
  max-height: 86dvh;
}

.filters__head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}

.filters__title {
  font-size: var(--fs-4);
  font-weight: 600;
}

.filters__optional {
  margin-left: auto;
  font-size: var(--fs-1);
  color: var(--text-3);
}

.filters__hint {
  font-size: var(--fs-2);
  color: var(--text-3);
  margin: 0 0 16px;
}

.filters__scroll {
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
  flex: 1;
  min-height: 0;
  padding-right: 12px;
}

.filters .skeleton-chips {
  --skeleton-base: var(--raised);
  --skeleton-highlight: var(--raised-2);
}

.filters__label {
  margin-bottom: 10px;
}

.filters__genres {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filters__genre {
  padding: 8px 14px;
  border-radius: var(--radius-pill);
  background: var(--raised);
  border: 1px solid transparent;
  color: var(--text-2);
  font-size: var(--fs-2);
}

.filters__genre--on {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--on-accent);
  font-weight: 600;
}

.filters__row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
}

.filters__value {
  font-size: var(--fs-3);
  font-weight: 600;
  color: var(--text);
}

.filters__segment {
  display: flex;
  background: var(--bg);
  border: 1px solid transparent;
  border-radius: var(--radius-m);
  padding: 4px;
  gap: 4px;
}

.filters__segment-btn {
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
  color: var(--text-body);
}

.filters__segment-btn--on {
  background: var(--accent);
  color: var(--on-accent);
  font-weight: 600;
}

/* :active bubbles up from the nested stepper buttons */
.filters__segment-btn:active {
  transform: none;
}

.filters__mini-stepper {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.filters__mini-step {
  width: 20px;
  height: 20px;
  border-radius: var(--radius-xs);
  border: none;
  background: rgba(0, 0, 0, 0.25);
  color: inherit;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.filters__foot {
  flex: none;
  display: flex;
  gap: 10px;
  padding-top: 14px;
  border-top: 1px solid var(--border-soft);
  margin-top: 14px;
}

.filters__reset {
  width: 96px;
  height: 52px;
  background: var(--raised);
  border: 1px solid transparent;
  color: var(--text-2);
  border-radius: var(--radius-l);
  font-weight: 600;
  font-size: var(--fs-3);
}

.filters__apply {
  flex: 1;
  height: 52px;
  background: var(--accent);
  color: var(--on-accent);
  border: none;
  border-radius: var(--radius-l);
  font-weight: 600;
  font-size: var(--fs-3);
}
</style>
