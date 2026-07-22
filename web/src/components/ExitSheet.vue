<script setup lang="ts">
import { t } from '../i18n'
import * as store from '../store/session'
import ConfirmSheet from './ConfirmSheet.vue'

const emit = defineEmits<{ close: [] }>()

async function confirmExit() {
  emit('close')
  if (store.isHost.value) {
    await store.finish()
  } else {
    await store.leave()
  }
}
</script>

<template>
  <ConfirmSheet
    :title="store.isHost.value ? t('continue.finishTitle') : t('continue.leaveTitle')"
    :text="store.isHost.value ? t('exit.hostText') : t('continue.leaveText')"
    :confirm-label="store.isHost.value ? t('continue.finishConfirm') : t('swipe.leave')"
    :cancel-label="t('exit.keepSwiping')"
    @confirm="confirmExit"
    @close="emit('close')"
  />
</template>
