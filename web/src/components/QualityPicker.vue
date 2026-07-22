<script setup lang="ts">
import { ref } from 'vue'
import { t } from '../i18n'
import * as store from '../store/session'
import SettingRow from './SettingRow.vue'
import PickerSheet from './PickerSheet.vue'

defineProps<{ disabled?: boolean }>()

const open = ref(false)

function pick(id: number) {
  store.selectQualityProfile(id)
  open.value = false
}
</script>

<template>
  <template v-if="store.qualityProfiles.value.length > 1">
    <SettingRow
      :label="t('quality.label')"
      :value="store.qualityProfileName.value"
      :disabled="disabled"
      @open="open = true"
    />

    <PickerSheet
      v-if="open"
      :title="t('quality.title')"
      :hint="t('quality.hint')"
      :options="store.qualityProfiles.value"
      :selected="store.qualityProfileId.value"
      :layer="80"
      @pick="pick"
      @close="open = false"
    />
  </template>
</template>
