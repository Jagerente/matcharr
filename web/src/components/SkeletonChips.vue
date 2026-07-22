<script setup lang="ts">
import { computed } from 'vue'

const CHIP_WIDTHS = [96, 72, 112, 84, 64, 104]
const STAGGER = 0.09

const props = withDefaults(defineProps<{ count?: number; height?: number; wrap?: boolean }>(), {
  count: 4,
  height: 38,
  wrap: false,
})

const chips = computed(() =>
  Array.from({ length: props.count }, (_, i) => ({
    width: CHIP_WIDTHS[i % CHIP_WIDTHS.length] + 'px',
    delay: (i % CHIP_WIDTHS.length) * STAGGER + 's',
  })),
)
</script>

<template>
  <div class="skeleton-chips" :class="{ 'skeleton-chips--wrap': wrap }" aria-hidden="true">
    <span
      v-for="(chip, i) in chips"
      :key="i"
      class="skeleton skeleton-chips__chip"
      :style="{ width: chip.width, height: height + 'px', animationDelay: chip.delay }"
    ></span>
  </div>
</template>

<style>
.skeleton-chips {
  display: flex;
  gap: 8px;
  overflow: hidden;
}

.skeleton-chips--wrap {
  flex-wrap: wrap;
  overflow: visible;
}

.skeleton-chips__chip {
  flex: none;
  border-radius: var(--radius-pill);
}
</style>
