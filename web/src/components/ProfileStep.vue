<script setup lang="ts">
import { computed, ref } from 'vue'
import { t } from '../i18n'
import AppHeader from './AppHeader.vue'
import LanguagePicker from './LanguagePicker.vue'
import Icon from './Icon.vue'

const AVATARS = ['🦊', '🐼', '🦉', '🐙', '🐝', '🦁', '🐢', '🦆', '🐳', '🦩', '🐸', '🦄']

const props = withDefaults(
  defineProps<{ name: string; avatar: string; pending: boolean; ctaLabel?: string }>(),
  { ctaLabel: '' },
)

const cta = computed(() => props.ctaLabel || t('profile.continue'))
const emit = defineEmits<{ back: []; done: [name: string, avatar: string] }>()

const name = ref(props.name)
const avatar = ref(
  props.avatar || AVATARS[Math.floor(Math.random() * AVATARS.length)] || '🦊',
)

const ready = computed(() => name.value.trim().length > 0)
</script>

<template>
  <div class="profile">
    <AppHeader>
      <template #left>
        <button class="btn-icon" type="button" @click="emit('back')">
          <Icon name="arrow-left" :size="18" />
        </button>
      </template>
    </AppHeader>
    <div class="profile__body">
      <h1 class="profile__title">{{ t('profile.title') }}</h1>
      <p class="profile__hint">{{ t('profile.hint') }}</p>

      <div class="profile__preview">{{ avatar }}</div>

      <div class="profile__avatars">
        <button
          v-for="a in AVATARS"
          :key="a"
          type="button"
          class="profile__avatar"
          :class="{ 'profile__avatar--active': a === avatar }"
          @click="avatar = a"
        >
          {{ a }}
        </button>
      </div>

      <input
        v-model="name"
        class="profile__name"
        maxlength="24"
        :placeholder="t('profile.namePlaceholder')"
        @keyup.enter="ready && emit('done', name.trim(), avatar)"
      />

      <LanguagePicker class="profile__lang" />

      <button
        class="btn btn--primary profile__cta"
        type="button"
        :disabled="!ready || pending"
        @click="emit('done', name.trim(), avatar)"
      >
        {{ cta }}
      </button>
    </div>
  </div>
</template>

<style>
.profile {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.profile__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 12px 28px 28px;
}

.profile__title {
  font-size: var(--fs-6);
  font-weight: 600;
  margin: 0 0 8px;
}

.profile__hint {
  font-size: var(--fs-3);
  color: var(--text-2);
  margin: 0 0 24px;
}

.profile__preview {
  width: 88px;
  height: 88px;
  border-radius: var(--radius-plate);
  background: var(--raised);
  border: 1px solid var(--accent-border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 46px;
  margin: 0 auto 24px;
}

.profile__avatars {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 10px;
  margin-bottom: 28px;
}

.profile__avatar {
  aspect-ratio: 1;
  border-radius: var(--radius-m);
  background: var(--surface);
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.profile__avatar--active {
  background: var(--accent);
}

.profile__name {
  height: 54px;
  border-radius: var(--radius-l);
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 0 18px;
  font-size: var(--fs-3);
  margin-bottom: 10px;
}

.profile__lang {
  margin-bottom: auto;
}

.profile__name:focus {
  outline: none;
  border-color: var(--accent-border-strong);
}

.profile__cta {
  margin-top: 24px;
}
</style>
