<script setup lang="ts">
import { computed, ref } from 'vue'
import * as store from '../store/session'
import { useBackGuard } from '../nav'
import { t } from '../i18n'
import ConfirmSheet from './ConfirmSheet.vue'
import StatTiles from './StatTiles.vue'
import WaitingNote from './WaitingNote.vue'
import AppHeader from './AppHeader.vue'
import Icon from './Icon.vue'

const state = computed(() => store.state.value!)
const isHost = computed(() => store.isHost.value)
const voting = computed(() => state.value.status === 'voting')
const proposal = computed(() => state.value.continue)
const extendBy = ref(25)

const myVote = computed(() => {
  const id = store.you.value?.id
  if (!id || !proposal.value) return undefined
  return proposal.value.votes[id]
})

const votedCount = computed(() => (proposal.value ? Object.keys(proposal.value.votes).length : 0))
const canPropose = computed(() => !state.value.sourceExhausted)

const stats = computed(() => [
  {
    value: state.value.matches.length,
    label: t('stats.matches', { count: state.value.matches.length }),
    accent: true,
  },
  { value: state.value.nearMissCount, label: t('stats.nearMisses') },
  { value: state.value.deckSize, label: t('stats.seen') },
])

const confirmFinish = ref(false)
const confirmLeave = ref(false)

useBackGuard(() => {
  if (isHost.value) confirmFinish.value = true
  else confirmLeave.value = true
})

async function doFinish() {
  confirmFinish.value = false
  await store.finish()
}

async function doLeave() {
  confirmLeave.value = false
  await store.leave()
}
</script>

<template>
  <div class="continue">
    <AppHeader />
    <template v-if="voting && proposal">
      <div class="continue__body continue__body--center">
        <div class="label label--accent continue__kicker">
          {{ t('continue.suggests', { host: store.hostName() }) }}
        </div>
        <h1 class="continue__title">
          {{ t('continue.proposalTitle', { count: proposal.proposedLimit }) }}
        </h1>
        <p class="continue__sub">{{ t('continue.proposalSub') }}</p>

        <div v-if="isHost" class="continue__tally">
          <div class="continue__tally-bar">
            <div
              class="continue__tally-fill"
              :style="{ width: (votedCount / state.participants.length) * 100 + '%' }"
            ></div>
          </div>
          <span class="continue__tally-count">
            {{ t('continue.tally', { voted: votedCount, total: state.participants.length }) }}
          </span>
        </div>

        <div class="continue__votes">
          <div
            v-for="p in state.participants"
            :key="p.id"
            class="continue__vote-row"
            :class="{ 'continue__vote-row--pending': proposal.votes[p.id] === undefined }"
          >
            <span class="continue__avatar">{{ p.avatar || '🙂' }}</span>
            <span class="continue__vote-name">
              {{ p.name }}
              <span v-if="p.id === store.you.value?.id" class="continue__you">
                {{ t('lobby.youBadge') }}
              </span>
            </span>
            <span v-if="proposal.votes[p.id]" class="continue__vote-yes">
              <Icon name="check" :size="13" /> {{ t('continue.keepGoing') }}
            </span>
            <span v-else class="continue__vote-wait">{{ t('continue.deciding') }}</span>
          </div>
        </div>
      </div>

      <div class="continue__foot">
        <template v-if="myVote === undefined">
          <button class="btn btn--primary" type="button" :disabled="store.busy.value" @click="store.voteContinue(true)">
            {{ t('continue.keepGoing') }}
          </button>
          <button class="btn btn--secondary" type="button" :disabled="store.busy.value" @click="store.voteContinue(false)">
            {{ t('continue.enough') }}
          </button>
        </template>
        <template v-else-if="isHost">
          <button class="btn btn--secondary" type="button" :disabled="store.busy.value" @click="store.cancelProposal()">
            {{ t('continue.cancelProposal') }}
          </button>
          <button class="btn btn--secondary" type="button" @click="confirmFinish = true">
            {{ t('swipe.endNow') }}
          </button>
        </template>
        <p v-else class="continue__waiting-note">{{ t('continue.voted') }}</p>
      </div>
    </template>

    <template v-else>
      <div class="continue__body">
        <div class="continue__hero">
          <div class="hero-glyph">🎞️</div>
          <h1 class="continue__title">
            {{ state.sourceExhausted ? t('continue.exhaustedTitle') : t('continue.deckTitle') }}
          </h1>
          <p class="continue__sub">
            {{
              state.sourceExhausted
                ? t('continue.exhaustedSub')
                : isHost
                  ? t('continue.hostSub')
                  : t('continue.guestSub', { host: store.hostName() })
            }}
          </p>
        </div>

        <StatTiles class="continue__stats" :stats="stats" />

        <template v-if="isHost && canPropose">
          <div class="label continue__label">{{ t('continue.addLabel') }}</div>
          <div class="continue__amounts">
            <button
              v-for="n in [10, 25, 50]"
              :key="n"
              type="button"
              class="continue__amount"
              :class="{ 'continue__amount--on': extendBy === n }"
              @click="extendBy = n"
            >
              +{{ n }}
            </button>
          </div>
          <p class="continue__note">{{ t('continue.note') }}</p>
        </template>

        <WaitingNote v-if="!isHost">{{ t('continue.waitHost') }}</WaitingNote>
      </div>

      <div class="continue__foot">
        <template v-if="isHost">
          <button
            v-if="canPropose"
            class="btn btn--primary"
            type="button"
            :disabled="store.busy.value"
            @click="store.propose(extendBy)"
          >
            {{ t('continue.propose', { count: extendBy }) }}
          </button>
          <button
            class="btn btn--secondary"
            type="button"
            :disabled="store.busy.value"
            @click="confirmFinish = true"
          >
            {{ t('continue.seeResults') }}
          </button>
        </template>
        <button v-else class="btn btn--secondary" type="button" @click="confirmLeave = true">
          {{ t('swipe.leave') }}
        </button>
      </div>
    </template>

    <ConfirmSheet
      v-if="confirmFinish"
      :title="t('continue.finishTitle')"
      :text="t('continue.finishText')"
      :confirm-label="t('continue.finishConfirm')"
      :cancel-label="t('continue.finishCancel')"
      @confirm="doFinish"
      @close="confirmFinish = false"
    />

    <ConfirmSheet
      v-if="confirmLeave"
      :title="t('continue.leaveTitle')"
      :text="t('continue.leaveText')"
      :confirm-label="t('swipe.leave')"
      :cancel-label="t('continue.leaveCancel')"
      @confirm="doLeave"
      @close="confirmLeave = false"
    />
  </div>
</template>

<style>
.continue {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.continue__body {
  flex: 1;
  overflow-y: auto;
  padding: 24px 24px 0;
}

.continue__body--center {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.continue__hero {
  text-align: center;
  margin-bottom: 24px;
}

.continue__kicker {
  margin-bottom: 14px;
  text-align: center;
}

.continue__title {
  font-size: var(--fs-5);
  font-weight: 600;
  margin: 0 0 8px;
  text-align: center;
  line-height: 1.2;
}

.continue__sub {
  font-size: var(--fs-3);
  color: var(--text-2);
  margin: 0 0 24px;
  text-align: center;
  line-height: 1.5;
}

.continue__stats {
  margin-bottom: 20px;
}

.continue__label {
  margin-bottom: 10px;
}

.continue__amounts {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
}

.continue__amount {
  flex: 1;
  text-align: center;
  padding: 14px 0;
  border-radius: var(--radius-m);
  background: var(--surface);
  border: 1px solid transparent;
  color: var(--text);
  font-weight: 600;
  font-size: var(--fs-3);
}

.continue__amount--on {
  background: var(--raised);
  border-color: var(--accent-border-strong);
}

.continue__note {
  font-size: var(--fs-2);
  color: var(--text-3);
  line-height: 1.5;
  margin: 0;
  text-align: center;
}

.continue__tally {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.continue__tally-bar {
  flex: 1;
  height: 6px;
  background: var(--raised);
  border-radius: var(--radius-pill);
  overflow: hidden;
}

.continue__tally-fill {
  height: 100%;
  background: var(--online);
  transition: width var(--t-screen) var(--ease);
}

.continue__tally-count {
  font-size: var(--fs-1);
  color: var(--text-2);
}

.continue__votes {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 8px;
}

.continue__vote-row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--surface);
  border-radius: var(--radius-m);
  padding: 11px 14px;
}

.continue__vote-row--pending {
  border: 1px solid var(--accent-line);
}

.continue__avatar {
  font-size: 22px;
}

.continue__vote-name {
  font-size: var(--fs-3);
  flex: 1;
}

.continue__you {
  font-size: var(--fs-1);
  font-weight: 600;
  color: var(--text-2);
}

.continue__vote-yes {
  font-size: var(--fs-2);
  color: var(--online);
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 5px;
}

.continue__vote-wait {
  font-size: var(--fs-2);
  color: var(--text-3);
}

.continue__waiting-note {
  text-align: center;
  color: var(--text-2);
  font-size: var(--fs-2);
  margin: 0;
}

.continue__foot {
  flex: none;
  padding: 14px 24px 26px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
