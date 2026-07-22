<script setup lang="ts">
import * as store from '../store/session'
import { useBackLayer } from '../nav'
import { t } from '../i18n'
import AppHeader from './AppHeader.vue'

useBackLayer(() => store.dismissNotice())
</script>

<template>
  <div class="end-notice">
    <AppHeader />
    <div class="end-notice__body">
      <div class="hero-glyph">🚪</div>
      <h1 class="end-notice__title">
        {{
          store.notice.value?.kind === 'removed'
            ? t('endNotice.removed')
            : t('endNotice.ended', {
                host: store.notice.value?.host ?? t('session.defaultHost'),
              })
        }}
      </h1>
      <p class="end-notice__text">{{ t('endNotice.text') }}</p>
    </div>
    <div class="end-notice__foot">
      <button class="btn btn--primary" type="button" @click="store.dismissNotice()">
        {{ t('results.back') }}
      </button>
    </div>
  </div>
</template>

<style>
.end-notice {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.end-notice__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 0 28px;
}

.end-notice__title {
  font-size: var(--fs-5);
  font-weight: 600;
  margin: 0 0 10px;
  line-height: 1.25;
}

.end-notice__text {
  font-size: var(--fs-2);
  color: var(--text-2);
  line-height: 1.55;
  margin: 0;
}

.end-notice__foot {
  flex: none;
  padding: 0 28px 30px;
}
</style>
