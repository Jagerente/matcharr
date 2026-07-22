<script setup lang="ts">
import Sheet from './Sheet.vue'

defineProps<{
  title: string
  text: string
  confirmLabel: string
  cancelLabel: string
}>()

const emit = defineEmits<{ confirm: []; close: [] }>()
</script>

<template>
  <Sheet class="confirm-sheet" @close="emit('close')">
    <h2 class="confirm-sheet__title">{{ title }}</h2>
    <p class="confirm-sheet__text">{{ text }}</p>
    <button class="btn btn--secondary confirm-sheet__confirm" type="button" @click="emit('confirm')">
      {{ confirmLabel }}
    </button>
    <button class="btn btn--primary" type="button" @click="emit('close')">
      {{ cancelLabel }}
    </button>
  </Sheet>
</template>

<style>
.confirm-sheet__title {
  font-size: var(--fs-4);
  font-weight: 600;
  margin: 0 0 8px;
  text-align: center;
}

.confirm-sheet__text {
  font-size: var(--fs-2);
  color: var(--text-2);
  margin: 0 0 22px;
  text-align: center;
  line-height: 1.5;
}

.confirm-sheet__confirm {
  height: 54px;
  margin-bottom: 10px;
}
</style>
