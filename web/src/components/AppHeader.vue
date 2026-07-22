<script setup lang="ts">
import Wordmark from './Wordmark.vue'
</script>

<template>
  <header class="app-header">
    <div class="app-header__side app-header__side--left">
      <slot name="left" />
    </div>
    <Wordmark class="app-header__brand" />
    <div class="app-header__side app-header__side--right">
      <slot name="right" />
    </div>
  </header>
</template>

<style>
.app-header {
  flex: none;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: 8px;
  height: 56px;
  padding: 0 20px;
}

.app-header__brand {
  font-size: var(--fs-6);
  justify-self: center;
}

.app-header__side {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.app-header__side--left {
  justify-self: start;
}

.app-header__side--right {
  justify-self: end;
}
</style>
