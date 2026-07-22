<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import Icon from './Icon.vue'

const props = withDefaults(defineProps<{ message: string; duration?: number }>(), {
  duration: 5000,
})
const emit = defineEmits<{ close: [] }>()

let timer: number | undefined

onMounted(() => {
  timer = window.setTimeout(() => emit('close'), props.duration)
})

onBeforeUnmount(() => window.clearTimeout(timer))
</script>

<template>
  <div class="error-toast" role="alert" @click="emit('close')">
    <span class="error-toast__ring">
      <svg class="error-toast__ring-svg" viewBox="0 0 36 36">
        <circle class="error-toast__ring-track" cx="18" cy="18" r="15.5" />
        <circle
          class="error-toast__ring-fill"
          cx="18"
          cy="18"
          r="15.5"
          :style="{ animationDuration: duration + 'ms' }"
        />
      </svg>
      <Icon name="alert" :size="14" class="error-toast__icon" />
    </span>
    <span class="error-toast__message">{{ message }}</span>
  </div>
</template>

<style>
.error-toast {
  position: absolute;
  top: 44px;
  left: 20px;
  right: 20px;
  z-index: 89;
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--raised);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-m);
  padding: 10px 14px;
  color: var(--text);
  font-size: var(--fs-2);
  line-height: 1.4;
  cursor: pointer;
  box-shadow: var(--shadow-toast);
  animation: fade-rise var(--t-item) var(--ease);
}

.error-toast__ring {
  position: relative;
  width: 30px;
  height: 30px;
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.error-toast__ring-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.error-toast__ring-track {
  fill: none;
  stroke: rgba(255, 255, 255, 0.12);
  stroke-width: 2.5;
}

.error-toast__ring-fill {
  fill: none;
  stroke: var(--accent);
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-dasharray: 97.4;
  stroke-dashoffset: 0;
  transform: rotate(-90deg);
  transform-origin: center;
  animation-name: toast-countdown;
  animation-timing-function: linear;
  animation-fill-mode: forwards;
}

@keyframes toast-countdown {
  from {
    stroke-dashoffset: 0;
  }
  to {
    stroke-dashoffset: 97.4;
  }
}

.error-toast__icon {
  color: var(--accent);
  position: relative;
}

.error-toast__message {
  flex: 1;
  min-width: 0;
}
</style>
