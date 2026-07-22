<script setup lang="ts">
import { computed } from 'vue'
import { t } from '../i18n'
import Sheet from './Sheet.vue'
import BrandMark from './BrandMark.vue'
import Wordmark from './Wordmark.vue'
import Icon from './Icon.vue'

const emit = defineEmits<{ close: [] }>()

const REPO_URL = 'https://github.com/Jagerente/matcharr'
const TMDB_URL = 'https://www.themoviedb.org'

// the wordmark is markup, so the title is split around its placeholder
// instead of being interpolated as text
const title = computed(() => {
  const [before = '', after = ''] = t('about.title').split('{name}')
  return { before, after }
})
</script>

<template>
  <Sheet class="about-sheet" @close="emit('close')">
    <BrandMark class="about-sheet__logo" :size="72" />
    <h2 class="about-sheet__title">{{ title.before }}<Wordmark />{{ title.after }}</h2>
    <p class="about-sheet__text">{{ t('about.text') }}</p>
    <p class="about-sheet__by">{{ t('about.madeBy') }}</p>
    <a
      class="btn btn--secondary about-sheet__link"
      :href="REPO_URL"
      target="_blank"
      rel="noopener noreferrer"
    >
      <Icon name="github" :size="18" />
      {{ t('about.github') }}
    </a>
    <button class="btn btn--primary" type="button" @click="emit('close')">
      {{ t('about.close') }}
    </button>
    <a
      class="about-sheet__tmdb"
      :href="TMDB_URL"
      target="_blank"
      rel="noopener noreferrer"
    >
      {{ t('about.tmdb') }}
    </a>
  </Sheet>
</template>

<style>
.about-sheet {
  text-align: center;
}

.about-sheet__logo {
  display: block;
  margin: 0 auto 10px;
}

.about-sheet__title {
  font-size: var(--fs-4);
  font-weight: 600;
  margin: 0 0 8px;
}

.about-sheet__text {
  font-size: var(--fs-2);
  color: var(--text-2);
  margin: 0 0 16px;
  line-height: 1.5;
}

.about-sheet__by {
  font-size: var(--fs-2);
  font-weight: 600;
  color: var(--text);
  margin: 0 0 22px;
}

.about-sheet__link {
  height: 54px;
  margin-bottom: 10px;
  text-decoration: none;
}

.about-sheet__tmdb {
  display: block;
  font-size: var(--fs-1);
  color: var(--text-3);
  margin-top: 16px;
  line-height: 1.4;
  text-decoration: none;
}
</style>
