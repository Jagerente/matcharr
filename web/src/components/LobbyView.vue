<script setup lang="ts">
import { computed, ref } from 'vue'
import { shareLink } from '../api/client'
import { copyText } from '../clipboard'
import { t } from '../i18n'
import * as store from '../store/session'
import { useBackGuard } from '../nav'
import QrSheet from './QrSheet.vue'
import ConfirmSheet from './ConfirmSheet.vue'
import WaitingNote from './WaitingNote.vue'
import AppHeader from './AppHeader.vue'
import Spinner from './Spinner.vue'
import Icon from './Icon.vue'

const state = computed(() => store.state.value!)
const isHost = computed(() => store.isHost.value)
const link = computed(() => shareLink(state.value.code))

const copied = ref(false)
const copyFailed = ref(false)
const qrOpen = ref(false)
const kickTarget = ref<{ id: string; name: string } | null>(null)
const cancelOpen = ref(false)
const leaveOpen = ref(false)

useBackGuard(() => {
  if (isHost.value) cancelOpen.value = true
  else leaveOpen.value = true
})

async function copyLink() {
  const ok = await copyText(link.value)
  copied.value = ok
  copyFailed.value = !ok
  if (!ok) qrOpen.value = true // the QR sheet shows the selectable link
  setTimeout(() => {
    copied.value = false
    copyFailed.value = false
  }, 1800)
}

async function confirmKick() {
  const target = kickTarget.value
  kickTarget.value = null
  if (target) {
    await store.kick(target.id)
  }
}

async function confirmCancel() {
  cancelOpen.value = false
  await store.finish()
}

async function confirmLeave() {
  leaveOpen.value = false
  await store.leave()
}

const canStart = computed(() => state.value.participants.length >= 2)
</script>

<template>
  <div class="lobby">
    <AppHeader />
    <div class="lobby__scroll">
      <div class="lobby__card">
        <div class="label">{{ t('lobby.code') }}</div>
        <div class="lobby__code">{{ state.code }}</div>
        <div class="lobby__share">
          <button
            type="button"
            class="lobby__share-btn"
            :class="{ 'lobby__share-btn--ok': copied }"
            @click="copyLink"
          >
            <Icon v-if="copied" name="check" :size="15" />
            {{ copied ? t('lobby.copied') : copyFailed ? t('lobby.showLink') : t('lobby.copy') }}
          </button>
          <button type="button" class="lobby__share-btn" @click="qrOpen = true">
            <Icon name="qr" :size="16" />QR
          </button>
        </div>
      </div>

      <div class="lobby__roster-head">
        <span class="lobby__roster-title">{{ t('lobby.roster') }}</span>
        <span class="lobby__roster-count">
          {{ t('lobby.joined', { count: state.participants.length }) }}
        </span>
      </div>

      <div class="lobby__roster">
        <div v-for="p in state.participants" :key="p.id" class="lobby__row">
          <span class="lobby__avatar">{{ p.avatar || '🙂' }}</span>
          <span class="lobby__name" :class="{ 'lobby__name--you': p.id === store.you.value?.id }">
            {{ p.name }}
          </span>
          <span v-if="p.isHost" class="lobby__host-badge">{{ t('lobby.hostBadge') }}</span>
          <span v-else-if="p.id === store.you.value?.id" class="lobby__you">
            {{ t('lobby.youBadge') }}
          </span>
          <span class="lobby__dot" :class="{ 'lobby__dot--online': p.online }"></span>
          <button
            v-if="isHost && !p.isHost"
            type="button"
            class="lobby__kick"
            @click="kickTarget = { id: p.id, name: p.name }"
          >
            <Icon name="user-minus" :size="17" />
          </button>
        </div>
      </div>
    </div>

    <div class="lobby__foot">
      <template v-if="isHost">
        <button
          class="btn btn--primary"
          type="button"
          :disabled="!canStart || store.busy.value"
          @click="store.start()"
        >
          <Spinner v-if="store.busy.value" />
          <template v-else>{{ canStart ? t('lobby.start') : t('lobby.waitMore') }}</template>
        </button>
        <button class="btn btn--secondary" type="button" @click="cancelOpen = true">
          {{ t('lobby.cancel') }}
        </button>
      </template>
      <template v-else>
        <WaitingNote class="lobby__hold">
          {{ t('lobby.waitingHost', { host: store.hostName() }) }}
        </WaitingNote>
        <button class="btn btn--secondary" type="button" @click="store.leave()">
          {{ t('lobby.leave') }}
        </button>
      </template>
    </div>

    <QrSheet v-if="qrOpen" :link="link" @close="qrOpen = false" />

    <ConfirmSheet
      v-if="kickTarget"
      :title="t('lobby.kickTitle', { name: kickTarget.name })"
      :text="t('lobby.kickText')"
      :confirm-label="t('lobby.kickConfirm')"
      :cancel-label="t('lobby.kickCancel')"
      @confirm="confirmKick"
      @close="kickTarget = null"
    />

    <ConfirmSheet
      v-if="cancelOpen"
      :title="t('lobby.cancelTitle')"
      :text="t('lobby.cancelText')"
      :confirm-label="t('lobby.cancelConfirm')"
      :cancel-label="t('lobby.cancelCancel')"
      @confirm="confirmCancel"
      @close="cancelOpen = false"
    />

    <ConfirmSheet
      v-if="leaveOpen"
      :title="t('lobby.leaveTitle')"
      :text="t('lobby.leaveText')"
      :confirm-label="t('lobby.leaveConfirm')"
      :cancel-label="t('lobby.leaveCancel')"
      @confirm="confirmLeave"
      @close="leaveOpen = false"
    />
  </div>
</template>

<style>
.lobby {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  position: relative;
}

.lobby__scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 0;
}

.lobby__card {
  background: var(--surface);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-xl);
  padding: 20px;
  text-align: center;
  margin-bottom: 18px;
}

.lobby__code {
  font-size: var(--fs-7);
  font-weight: 700;
  letter-spacing: 0.28em;
  padding-left: 0.28em;
  margin: 12px 0 16px;
}

.lobby__share {
  display: flex;
  gap: 10px;
}

.lobby__share-btn {
  flex: 1;
  height: 44px;
  border-radius: var(--radius-m);
  background: var(--raised);
  border: 1px solid transparent;
  color: var(--text);
  font-weight: 600;
  font-size: var(--fs-2);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background var(--t-item), border-color var(--t-item), color var(--t-item);
}

.lobby__share-btn--ok {
  background: var(--online-soft);
  border-color: var(--online-border);
  color: var(--online);
}

.lobby__roster-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 12px;
}

.lobby__roster-title {
  font-size: var(--fs-3);
  font-weight: 600;
}

.lobby__roster-count {
  font-size: var(--fs-2);
  color: var(--text-2);
}

.lobby__roster {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 16px;
}

.lobby__row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--surface);
  border-radius: var(--radius-m);
  padding: 10px 14px;
  animation: fade-rise var(--t-item) var(--ease);
}

.lobby__avatar {
  font-size: 24px;
}

.lobby__name {
  font-size: var(--fs-3);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lobby__name--you {
  font-weight: 600;
}

.lobby__host-badge {
  font-size: var(--fs-1);
  font-weight: 600;
  line-height: 1.2;
  color: var(--accent-text);
  background: var(--accent-soft);
  padding: 3px 8px;
  border-radius: var(--radius-pill);
}

.lobby__you {
  font-size: var(--fs-1);
  font-weight: 600;
  color: var(--text-2);
}

.lobby__dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-pill);
  background: var(--muted);
}

.lobby__dot--online {
  background: var(--online);
  animation: pulse 1.6s infinite;
}

.lobby__kick {
  background: none;
  border: none;
  color: var(--muted);
  padding: 0 2px;
}

.lobby__foot {
  flex: none;
  padding: 14px 24px 26px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.lobby__hold {
  margin-bottom: 8px;
}
</style>
