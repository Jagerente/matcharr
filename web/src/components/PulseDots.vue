<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 8 })
</script>

<template>
  <span class="pulse-dots" :style="{ '--dot-size': size + 'px' }">
    <span></span><span></span><span></span>
  </span>
</template>

<style>
.pulse-dots {
  display: inline-flex;
  gap: 6px;
  flex: none;
}

.pulse-dots span {
  width: var(--dot-size);
  height: var(--dot-size);
  border-radius: var(--radius-pill);
  background: var(--accent);
  animation: pulse 1.2s infinite;
}

.pulse-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.pulse-dots span:nth-child(3) {
  animation-delay: 0.4s;
}
</style>
