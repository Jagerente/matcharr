<script setup lang="ts" generic="T extends string | number">
import Sheet from './Sheet.vue'
import Icon from './Icon.vue'

defineProps<{
  title: string
  hint?: string
  options: readonly { id: T; name: string }[]
  selected: T
  layer?: number
}>()
const emit = defineEmits<{ pick: [id: T]; close: [] }>()
</script>

<template>
  <Sheet class="picker-sheet" :layer="layer" @close="emit('close')">
    <div class="picker-sheet__title">{{ title }}</div>
    <p v-if="hint" class="picker-sheet__hint">{{ hint }}</p>
    <div class="picker-sheet__list">
      <button
        v-for="o in options"
        :key="o.id"
        type="button"
        class="picker-sheet__opt"
        :class="{ 'picker-sheet__opt--on': o.id === selected }"
        @click="emit('pick', o.id)"
      >
        <span class="picker-sheet__opt-name">{{ o.name }}</span>
        <Icon v-if="o.id === selected" name="check" :size="16" class="picker-sheet__check" />
      </button>
    </div>
  </Sheet>
</template>

<style>
.picker-sheet {
  display: flex;
  flex-direction: column;
  max-height: 80dvh;
}

.picker-sheet__title {
  font-size: var(--fs-4);
  font-weight: 600;
}

.picker-sheet__hint {
  font-size: var(--fs-2);
  color: var(--text-3);
  margin: 4px 0 14px;
  line-height: 1.45;
}

.picker-sheet__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
  min-height: 0;
}

.picker-sheet__opt {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--raised);
  border: 1px solid transparent;
  border-radius: var(--radius-s);
  padding: 12px 14px;
  font-size: var(--fs-2);
  color: var(--text-body);
  text-align: left;
}

.picker-sheet__opt--on {
  border-color: var(--accent);
  color: var(--text);
  font-weight: 600;
}

.picker-sheet__opt-name {
  flex: 1;
}

.picker-sheet__check {
  flex: none;
  color: var(--accent);
}
</style>
