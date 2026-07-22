<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import * as store from '../store/session'
import { useBackLayer } from '../nav'
import { t } from '../i18n'
import { ApiError, type Settings } from '../api/client'
import LanguagePicker from './LanguagePicker.vue'
import JoinCodeStep from './JoinCodeStep.vue'
import ProfileStep from './ProfileStep.vue'
import CreateStep from './CreateStep.vue'
import ErrorToast from './ErrorToast.vue'
import PulseDots from './PulseDots.vue'
import Icon from './Icon.vue'
import BrandMark from './BrandMark.vue'
import Wordmark from './Wordmark.vue'
import AppHeader from './AppHeader.vue'
import AboutSheet from './AboutSheet.vue'

type Step = 'welcome' | 'code' | 'profile' | 'create' | 'joining'
const step = ref<Step>('welcome')
const profileNext = ref<'create' | 'join' | 'save'>('save')
const code = ref('')
const profile = ref(store.savedProfile())
const pending = ref(false)
const formError = ref('')
const codeError = ref('')
const joinViaLink = ref(false)
const aboutOpen = ref(false)

const hasProfile = computed(() => profile.value.name.trim().length > 0)

// The welcome screen owns no invite, so the hash drops with it: pasting the
// same link again then still counts as a new one.
function toWelcome() {
  step.value = 'welcome'
  store.clearSessionLink()
}

useBackLayer(toWelcome, () => step.value !== 'welcome')

// Taken, not read: an invite pasted twice has to start the join twice.
function enterViaLink(linkCode: string) {
  store.invite.value = ''
  joinViaLink.value = true
  code.value = linkCode
  if (hasProfile.value) {
    void doJoin()
  } else {
    profileNext.value = 'join'
    step.value = 'profile'
  }
}

if (store.invite.value) enterViaLink(store.invite.value)

watch(store.invite, (next) => {
  if (next) enterViaLink(next)
})

function choose(next: 'create' | 'join') {
  if (next === 'join') {
    codeError.value = ''
    step.value = 'code'
    return
  }
  if (hasProfile.value) {
    step.value = 'create'
  } else {
    profileNext.value = 'create'
    step.value = 'profile'
  }
}

function editProfile() {
  profileNext.value = 'save'
  step.value = 'profile'
}

function onCode(entered: string) {
  joinViaLink.value = false
  code.value = entered
  if (hasProfile.value) {
    void doJoin()
  } else {
    profileNext.value = 'join'
    step.value = 'profile'
  }
}

function onProfile(name: string, avatar: string) {
  profile.value = { name, avatar }
  store.updateProfile(name, avatar)
  switch (profileNext.value) {
    case 'create':
      step.value = 'create'
      break
    case 'join':
      void doJoin()
      break
    default:
      toWelcome()
  }
}

async function doJoin() {
  // On the code step the button carries the spinner, so a fast failure just
  // re-enables it in place instead of flashing the full-screen joining view.
  if (step.value !== 'code') step.value = 'joining'
  codeError.value = ''
  pending.value = true
  try {
    await store.join(code.value, profile.value.name, profile.value.avatar)
  } catch (e) {
    if (e instanceof ApiError && e.reason === 'session_not_found') {
      store.clearSessionLink()
      if (joinViaLink.value) {
        code.value = ''
        codeError.value = t('join.deadLink')
      } else {
        codeError.value = t('join.badCode')
      }
    } else {
      codeError.value = store.errorText(e)
    }
    step.value = 'code'
  } finally {
    pending.value = false
  }
}

async function onCreate(settings: Settings) {
  formError.value = ''
  pending.value = true
  try {
    await store.create(profile.value.name, profile.value.avatar, settings)
  } catch (e) {
    formError.value = store.errorText(e)
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <div class="entry">
    <ErrorToast v-if="formError" :key="formError" :message="formError" @close="formError = ''" />

    <div v-if="step === 'welcome'" class="welcome">
      <div class="welcome__top">
        <LanguagePicker variant="pill" />
      </div>
      <div class="welcome__body">
        <BrandMark class="welcome__logo" :size="240" />
        <h1 class="welcome__title"><Wordmark /></h1>
        <p class="welcome__tagline">{{ t('welcome.tagline') }}</p>
        <button class="btn btn--primary welcome__cta" type="button" @click="choose('create')">
          {{ t('welcome.create') }}
        </button>
        <button class="btn btn--secondary" type="button" @click="choose('join')">
          {{ t('welcome.join') }}
        </button>
        <button v-if="hasProfile" class="welcome__profile" type="button" @click="editProfile">
          <span class="welcome__profile-avatar">{{ profile.avatar || '🙂' }}</span>
          <span class="welcome__profile-name">{{ profile.name }}</span>
          <Icon name="chevron-right" :size="15" class="welcome__profile-chev" />
        </button>
      </div>
      <div class="welcome__foot">
        <button class="welcome__about" type="button" @click="aboutOpen = true">
          <Icon name="info" :size="15" />
          {{ t('welcome.about') }}
        </button>
      </div>
      <AboutSheet v-if="aboutOpen" @close="aboutOpen = false" />
    </div>

    <JoinCodeStep
      v-else-if="step === 'code'"
      :initial="code"
      :error="codeError"
      :pending="pending"
      @back="toWelcome"
      @done="onCode"
    />

    <ProfileStep
      v-else-if="step === 'profile'"
      :name="profile.name"
      :avatar="profile.avatar"
      :pending="pending"
      :cta-label="profileNext === 'save' ? t('profile.save') : t('profile.continue')"
      @back="toWelcome"
      @done="onProfile"
    />

    <div v-else-if="step === 'joining'" class="entry-joining">
      <AppHeader />
      <div class="entry-joining__body">
        <PulseDots :size="10" />
        <p class="entry-joining__text">{{ t('join.joining', { code }) }}</p>
      </div>
    </div>

    <CreateStep v-else :pending="pending" @back="toWelcome" @create="onCreate" />
  </div>
</template>

<style>
.entry {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.welcome {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.welcome__top {
  flex: none;
  display: flex;
  justify-content: flex-end;
  padding: 12px 20px 0;
}

.welcome__logo {
  flex: none;
  align-self: center;
  margin-bottom: 4px;
}

.welcome__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 0 28px 40px;
}

.welcome__title {
  font-size: var(--fs-7);
  margin: 0 0 12px;
}

.welcome__tagline {
  font-size: var(--fs-3);
  color: var(--text-2);
  margin: 0 0 44px;
}

.welcome__cta {
  margin-bottom: 14px;
}

.welcome__profile {
  margin-top: 28px;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--surface);
  border: 1px solid transparent;
  border-radius: var(--radius-pill);
  padding: 8px 14px 8px 10px;
  color: var(--text-2);
  font-size: var(--fs-2);
  font-weight: 600;
}

.welcome__profile-avatar {
  font-size: 20px;
}

.welcome__profile-name {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.welcome__profile-chev {
  color: var(--muted);
}

.welcome__foot {
  flex: none;
  display: flex;
  justify-content: center;
  padding: 0 28px 20px;
}

.welcome__about {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  background: none;
  border: none;
  padding: 8px 12px;
  color: var(--muted);
  font-size: var(--fs-2);
  font-weight: 600;
}

.entry-joining {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.entry-joining__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18px;
}

.entry-joining__text {
  margin: 0;
  color: var(--text-2);
  font-size: var(--fs-3);
}
</style>
