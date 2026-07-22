<script setup lang="ts">
import PulseDots from './PulseDots.vue'
</script>

<template>
  <div class="note waiting-note">
    <PulseDots />
    <span class="waiting-note__text"><slot /></span>
  </div>
</template>

<style>
.waiting-note__text {
  flex: 1;
}
</style>
