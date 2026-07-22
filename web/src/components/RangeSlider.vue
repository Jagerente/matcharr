<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    min: number
    max: number
    step?: number
    modelValue?: number
    from?: number
    to?: number
    range?: boolean
  }>(),
  { step: 1 },
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
  'update:from': [value: number]
  'update:to': [value: number]
}>()

const span = computed(() => props.max - props.min || 1)
const pct = (v: number) => ((v - props.min) / span.value) * 100

const fillStyle = computed(() => {
  if (props.range) {
    const a = pct(props.from ?? props.min)
    const b = pct(props.to ?? props.max)
    return { left: `${a}%`, width: `${b - a}%` }
  }
  return { left: '0%', width: `${pct(props.modelValue ?? props.min)}%` }
})

// When both thumbs meet, the from-thumb needs to sit on top or it becomes
// impossible to drag it back out from under the to-thumb.
const fromOnTop = computed(() => (props.from ?? props.min) >= (props.to ?? props.max))

function num(e: Event): number {
  return Number((e.target as HTMLInputElement).value)
}

function onSingle(e: Event) {
  emit('update:modelValue', num(e))
}

function onFrom(e: Event) {
  emit('update:from', Math.min(num(e), props.to ?? props.max))
}

function onTo(e: Event) {
  emit('update:to', Math.max(num(e), props.from ?? props.min))
}
</script>

<template>
  <div class="rng">
    <div class="rng__track">
      <div class="rng__fill" :style="fillStyle"></div>
    </div>

    <template v-if="range">
      <input
        class="rng__input"
        type="range"
        :min="min"
        :max="max"
        :step="step"
        :value="from"
        :style="{ zIndex: fromOnTop ? 5 : 3 }"
        @input="onFrom"
      />
      <input
        class="rng__input"
        type="range"
        :min="min"
        :max="max"
        :step="step"
        :value="to"
        @input="onTo"
      />
    </template>

    <input
      v-else
      class="rng__input rng__input--solo"
      type="range"
      :min="min"
      :max="max"
      :step="step"
      :value="modelValue"
      @input="onSingle"
    />
  </div>
</template>

<style>
.rng {
  position: relative;
  width: 100%;
  height: 24px;
  display: flex;
  align-items: center;
}

.rng__track {
  position: absolute;
  left: 0;
  right: 0;
  height: 5px;
  border-radius: var(--radius-pill);
  background: #fff;
}

.rng__fill {
  position: absolute;
  top: 0;
  bottom: 0;
  background: var(--accent);
  border-radius: var(--radius-pill);
}

.rng__input {
  position: absolute;
  left: 0;
  right: 0;
  width: 100%;
  height: 24px;
  margin: 0;
  background: none;
  pointer-events: none;
  z-index: 4;
  -webkit-appearance: none;
  appearance: none;
}

.rng__input--solo {
  pointer-events: auto;
}

.rng__input:focus-visible {
  outline: none;
}

.rng__input::-webkit-slider-runnable-track {
  -webkit-appearance: none;
  height: 24px;
  background: transparent;
}

.rng__input::-webkit-slider-thumb {
  -webkit-appearance: none;
  pointer-events: auto;
  width: 18px;
  height: 18px;
  margin-top: 3px;
  border-radius: var(--radius-pill);
  background: var(--accent);
  border: none;
  box-shadow: var(--shadow-rest);
  cursor: pointer;
}

.rng__input::-moz-range-track {
  height: 24px;
  background: transparent;
}

.rng__input::-moz-range-progress {
  background: transparent;
}

.rng__input::-moz-range-thumb {
  pointer-events: auto;
  width: 18px;
  height: 18px;
  border-radius: var(--radius-pill);
  background: var(--accent);
  border: none;
  box-shadow: var(--shadow-rest);
  cursor: pointer;
}

.rng__input:focus-visible::-webkit-slider-thumb {
  outline: 2px solid var(--accent-border);
  outline-offset: 2px;
}

.rng__input:focus-visible::-moz-range-thumb {
  outline: 2px solid var(--accent-border);
  outline-offset: 2px;
}
</style>
