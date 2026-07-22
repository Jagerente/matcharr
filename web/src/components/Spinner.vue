<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 18 })
</script>

<template>
  <span class="spinner" :style="{ width: size + 'px', height: size + 'px' }" aria-hidden="true"></span>
</template>

<style>
.spinner {
  display: inline-block;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: var(--radius-pill);
  opacity: 0.9;
  animation: spin 0.7s linear infinite;
}
</style>
