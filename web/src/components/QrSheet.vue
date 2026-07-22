<script setup lang="ts">
import { ref, watchEffect } from 'vue'
import QRCode from 'qrcode'
import { t } from '../i18n'
import Sheet from './Sheet.vue'

const props = defineProps<{ link: string }>()
const emit = defineEmits<{ close: [] }>()

const dataUrl = ref('')

watchEffect(async () => {
  try {
    dataUrl.value = await QRCode.toDataURL(props.link, {
      margin: 1,
      width: 260,
      color: { dark: '#f2eff4', light: '#1c1a2200' },
    })
  } catch {
    dataUrl.value = ''
  }
})
</script>

<template>
  <Sheet class="qr-sheet" @close="emit('close')">
    <h2 class="qr-sheet__title">{{ t('qr.title') }}</h2>
    <img v-if="dataUrl" class="qr-sheet__img" :src="dataUrl" :alt="t('qr.alt')" />
    <p class="qr-sheet__hint">{{ t('qr.hint') }}</p>
    <p class="qr-sheet__link">{{ link }}</p>
    <button class="btn btn--secondary" type="button" @click="emit('close')">
      {{ t('qr.back') }}
    </button>
  </Sheet>
</template>

<style>
.qr-sheet {
  text-align: center;
}

.qr-sheet__title {
  font-size: var(--fs-4);
  font-weight: 600;
  margin: 0 0 18px;
}

.qr-sheet__img {
  margin: 0 auto 18px;
  border-radius: var(--radius-m);
  background: var(--surface);
  padding: 12px;
}

.qr-sheet__hint {
  font-size: var(--fs-2);
  color: var(--text-3);
  margin: 0 0 6px;
}

.qr-sheet__link {
  margin: 0 0 20px;
  font-size: var(--fs-2);
  color: var(--text-2);
  overflow-wrap: anywhere;
  user-select: all;
}
</style>
