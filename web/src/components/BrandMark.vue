<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 84 })

// hash routing keeps the document path fixed, so the relative base resolves
// the same at the root and behind /matcharr
const src = `${import.meta.env.BASE_URL}favicon/icon.svg`
</script>

<template>
  <img class="brand-mark" :src="src" :width="size" :height="size" alt="" />
</template>

<style>
.brand-mark {
  flex: none;
}
</style>
