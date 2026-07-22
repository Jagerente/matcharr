<script setup lang="ts">
import { ref } from 'vue'
import { LOCALES, currentLocale, setLocale, t, type Locale } from '../i18n'
import SettingRow from './SettingRow.vue'
import PickerSheet from './PickerSheet.vue'
import Icon from './Icon.vue'

withDefaults(defineProps<{ variant?: 'row' | 'pill' }>(), { variant: 'row' })

const open = ref(false)

function pick(id: Locale) {
  setLocale(id)
  open.value = false
}
</script>

<template>
  <div>
    <button
      v-if="variant === 'pill'"
      type="button"
      class="lang-pill"
      :aria-label="t('language.label')"
      @click="open = true"
    >
      <Icon name="globe" :size="15" />{{ currentLocale.short }}
    </button>
    <SettingRow
      v-else
      icon="globe"
      :label="t('language.label')"
      :value="currentLocale.name"
      @open="open = true"
    />

    <PickerSheet
      v-if="open"
      :title="t('language.title')"
      :hint="t('language.hint')"
      :options="LOCALES"
      :selected="currentLocale.id"
      :layer="80"
      @pick="pick"
      @close="open = false"
    />
  </div>
</template>

<style>
.lang-pill {
  display: flex;
  align-items: center;
  gap: 7px;
  background: var(--surface);
  border: 1px solid transparent;
  border-radius: var(--radius-pill);
  padding: 7px 14px;
  color: var(--text-2);
  font-size: var(--fs-2);
  font-weight: 600;
}
</style>
