<script setup lang="ts">
import type { IconName } from '../icons'
import Icon from './Icon.vue'

defineProps<{ label: string; value: string; icon?: IconName; disabled?: boolean }>()
const emit = defineEmits<{ open: [] }>()
</script>

<template>
  <button type="button" class="setting-row" :disabled="disabled" @click="emit('open')">
    <Icon v-if="icon" :name="icon" :size="17" class="setting-row__icon" />
    <span class="label">{{ label }}</span>
    <span class="setting-row__value">{{ value }}</span>
    <Icon name="chevron-right" :size="16" class="setting-row__chev" />
  </button>
</template>

<style>
.setting-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  background: var(--surface);
  border: 1px solid transparent;
  border-radius: var(--radius-m);
  padding: 11px 14px;
  text-align: left;
}

.sheet .setting-row {
  background: var(--raised);
}

.setting-row:disabled {
  opacity: 0.5;
}

.setting-row__icon {
  color: var(--text-2);
}

.setting-row__value {
  flex: 1;
  font-size: var(--fs-3);
  color: var(--text);
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.setting-row__chev {
  flex: none;
  color: var(--muted);
}
</style>
