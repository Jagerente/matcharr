<script setup lang="ts">
defineProps<{
  stats: { value: string | number; label: string; accent?: boolean }[]
}>()
</script>

<template>
  <div class="stat-tiles">
    <div v-for="s in stats" :key="s.label" class="stat-tiles__tile">
      <div class="stat-tiles__value" :class="{ 'stat-tiles__value--accent': s.accent }">
        {{ s.value }}
      </div>
      <div class="stat-tiles__label">{{ s.label }}</div>
    </div>
  </div>
</template>

<style>
.stat-tiles {
  display: flex;
  gap: 10px;
  width: 100%;
}

.stat-tiles__tile {
  flex: 1;
  background: var(--surface);
  border-radius: var(--radius-m);
  padding: 16px 0;
  text-align: center;
}

.stat-tiles__value {
  font-size: var(--fs-5);
  font-weight: 700;
}

.stat-tiles__value--accent {
  color: var(--accent);
}

.stat-tiles__label {
  font-size: var(--fs-2);
  color: var(--text-body);
}
</style>
