<script setup lang="ts">
import { computed } from 'vue'
import { ICONS, type IconName } from '../icons'

const props = withDefaults(
  defineProps<{ name: IconName; size?: number; filled?: boolean }>(),
  { size: 18, filled: false },
)

const markup = computed(() => ICONS[props.name])
</script>

<template>
  <svg
    class="icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    :fill="filled ? 'currentColor' : 'none'"
    stroke="currentColor"
    stroke-width="2"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    v-html="markup"
  ></svg>
</template>

<style>
.icon {
  flex: none;
  display: inline-block;
  vertical-align: -0.15em;
}
</style>
