<script setup lang="ts">
import { computed, ref } from 'vue'
import { useBackLayer } from '../nav'

const props = withDefaults(defineProps<{ layer?: number }>(), { layer: 60 })
const emit = defineEmits<{ close: [] }>()

defineOptions({ inheritAttrs: false })

useBackLayer(() => emit('close'))

const CLOSE_THRESHOLD = 90

const dy = ref(0)
const dragging = ref(false)
let startY = 0

function onDown(e: PointerEvent) {
  dragging.value = true
  startY = e.clientY
  dy.value = 0
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
}

function onMove(e: PointerEvent) {
  if (!dragging.value) return
  dy.value = Math.max(0, e.clientY - startY)
}

function onUp() {
  if (!dragging.value) return
  dragging.value = false
  if (dy.value > CLOSE_THRESHOLD) {
    emit('close')
  }
  dy.value = 0
}

const sheetStyle = computed(() => ({
  zIndex: String(props.layer + 1),
  // A resting transform would make this sheet the containing block of any
  // fixed child, and a sheet opened from within would anchor to it instead
  // of the viewport.
  transform: dy.value ? `translateY(${dy.value}px)` : undefined,
  transition: dragging.value ? 'none' : 'transform var(--t-screen) var(--ease)',
}))
</script>

<template>
  <div>
    <div class="sheet-backdrop" :style="{ zIndex: String(layer) }" @click="emit('close')"></div>
    <div class="sheet" v-bind="$attrs" :style="sheetStyle">
      <div
        class="sheet__grab"
        @pointerdown="onDown"
        @pointermove="onMove"
        @pointerup="onUp"
        @pointercancel="onUp"
      >
        <div class="sheet__handle"></div>
      </div>
      <slot />
    </div>
  </div>
</template>
