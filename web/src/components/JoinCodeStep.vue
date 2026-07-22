<script setup lang="ts">
// The invisible input is full-size on purpose: zero-sized inputs derail
// mobile IMEs (caret jumps to start, backspace dies).
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { t } from '../i18n'
import AppHeader from './AppHeader.vue'
import Spinner from './Spinner.vue'
import Icon from './Icon.vue'

const LENGTH = 4

const props = defineProps<{ initial?: string; error?: string; pending?: boolean }>()
const emit = defineEmits<{ back: []; done: [code: string] }>()

const code = ref(clean(props.initial ?? ''))
const focused = ref(false)
const input = ref<HTMLInputElement | null>(null)
const boxes = ref<HTMLElement | null>(null)

// The focus event never comes for a programmatic focus on an input that
// already holds it, and the caret hangs on this flag.
function takeFocus() {
  input.value?.focus()
  focused.value = document.activeElement === input.value
}

const localError = ref(props.error ?? '')
watch(
  () => props.error,
  (e) => {
    localError.value = e ?? ''
    if (e) takeFocus()
  },
)
// A stale error would just confuse once they start fixing the code.
watch(code, () => (localError.value = ''))

const chars = computed(() => Array.from({ length: LENGTH }, (_, i) => code.value[i] ?? ''))
const complete = computed(() => code.value.length === LENGTH)
// A full code has nothing left to point at, so no single box is singled out:
// the group lights up as a whole instead of stranding a lit, caretless box.
const activeIndex = computed(() => (focused.value && !complete.value ? code.value.length : -1))

function clean(raw: string): string {
  return raw
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, '')
    .slice(0, LENGTH)
}

function setCode(next: string) {
  code.value = next
  if (next.length === LENGTH) submit()
}

function submit() {
  if (!complete.value || props.pending) return
  emit('done', code.value)
}

function onInput(e: Event) {
  const el = e.target as HTMLInputElement
  const next = clean(el.value)
  if (el.value !== next) {
    el.value = next
  }
  // caret pinned to the end, or mobile keyboards insert before it
  el.setSelectionRange(next.length, next.length)
  setCode(next)
}

// An invite link is pasted as often as the bare code it carries.
function onPaste(e: ClipboardEvent) {
  const text = e.clipboardData?.getData('text') ?? ''
  const next = clean(text.match(/#\/s\/([A-Za-z0-9]+)/)?.[1] ?? text)
  if (!next) return
  e.preventDefault()
  setCode(next)
}

function onFocus() {
  focused.value = true
  window.setTimeout(() => {
    boxes.value?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }, 300)
}

function onWindowKey(e: KeyboardEvent) {
  if (e.ctrlKey || e.metaKey || e.altKey) return
  // skip when the input has focus, it processes keys itself
  if (document.activeElement === input.value) return
  if (e.key === 'Enter') {
    submit()
    return
  }
  if (e.key === 'Backspace') {
    code.value = code.value.slice(0, -1)
    e.preventDefault()
    return
  }
  if (/^[a-zA-Z0-9]$/.test(e.key)) {
    setCode(clean(code.value + e.key))
    e.preventDefault()
  }
}

onMounted(() => {
  takeFocus()
  window.addEventListener('keydown', onWindowKey)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onWindowKey))
</script>

<template>
  <div class="join">
    <AppHeader>
      <template #left>
        <button class="btn-icon" type="button" @click="emit('back')">
          <Icon name="arrow-left" :size="18" />
        </button>
      </template>
    </AppHeader>
    <div class="join__body">
      <h1 class="join__title">{{ t('join.title') }}</h1>
      <p class="join__hint">{{ t('join.hint') }}</p>

      <label
        ref="boxes"
        class="join__boxes"
        :class="{
          'join__boxes--complete': complete,
          'join__boxes--error': localError !== '',
        }"
      >
        <input
          ref="input"
          class="join__input"
          :value="code"
          :maxlength="LENGTH"
          :aria-label="t('join.title')"
          autocapitalize="characters"
          autocomplete="one-time-code"
          autocorrect="off"
          spellcheck="false"
          enterkeyhint="go"
          @input="onInput"
          @paste="onPaste"
          @focus="onFocus"
          @blur="focused = false"
          @keyup.enter="submit"
        />
        <span
          v-for="(ch, i) in chars"
          :key="i"
          class="join__box"
          :class="{
            'join__box--filled': ch !== '',
            'join__box--active': i === activeIndex,
          }"
        >
          {{ ch }}
          <span v-if="i === activeIndex" class="join__caret"></span>
        </span>
      </label>

      <p v-if="localError" class="join__error" role="status">
        <Icon name="alert" :size="15" class="join__error-icon" />
        <span>{{ localError }}</span>
      </p>

      <button
        class="btn btn--primary"
        type="button"
        :disabled="!complete || pending"
        @click="submit"
      >
        <Spinner v-if="pending" />
        <template v-else>{{ t('join.cta') }}</template>
      </button>
    </div>
  </div>
</template>

<style>
.join {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.join__body {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 0 28px 60px;
}

.join__title {
  font-size: var(--fs-6);
  font-weight: 600;
  margin: 0 0 8px;
}

.join__hint {
  font-size: var(--fs-3);
  color: var(--text-2);
  margin: 0 0 32px;
}

.join__boxes {
  display: flex;
  gap: 12px;
  margin-bottom: 28px;
  position: relative;
  cursor: text;
}

.join__error {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--fs-2);
  color: var(--accent-text);
  line-height: 1.4;
  margin: -8px 0 22px;
}

.join__error-icon {
  flex: none;
  color: var(--accent);
}

.join__input {
  /* 16px font stops iOS zoom-on-focus */
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  opacity: 0;
  font-size: var(--fs-3);
  border: none;
  background: none;
  color: transparent;
  caret-color: transparent;
  z-index: 1;
}

.join__box {
  flex: 1;
  aspect-ratio: 1;
  border-radius: var(--radius-m);
  background: var(--surface);
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--fs-7);
  font-weight: 600;
  color: var(--text);
  transition:
    border-color var(--t-item) var(--ease),
    background-color var(--t-item) var(--ease),
    box-shadow var(--t-item) var(--ease);
}

.join__box--filled {
  border-color: var(--accent-border);
}

.join__box--active {
  border-color: var(--accent);
}

.join__boxes--complete .join__box {
  border-color: var(--accent-border-strong);
  box-shadow: var(--glow-hint);
}

.join__boxes--error .join__box {
  border-color: var(--accent);
  background: var(--accent-soft);
  box-shadow: none;
}

.join__boxes--error {
  animation: join-shake 300ms var(--ease);
}

.join__caret {
  width: 2px;
  height: 30px;
  background: var(--accent);
  margin-left: 2px;
  animation: pulse 1s infinite;
}

@keyframes join-shake {
  0%,
  100% {
    transform: translateX(0);
  }
  20% {
    transform: translateX(-6px);
  }
  40% {
    transform: translateX(5px);
  }
  60% {
    transform: translateX(-3px);
  }
  80% {
    transform: translateX(2px);
  }
}
</style>
