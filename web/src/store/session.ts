import { computed, ref, watch } from 'vue'
import * as api from '../api/client'
import { t, tReason } from '../i18n'
import type {
  Genre,
  Movie,
  PublicState,
  QualityProfile,
  SessionEvent,
  Settings,
  Verdict,
  You,
} from '../api/client'

const STORAGE_KEY = 'matcharr:session'
const NAME_KEY = 'matcharr:profile'
const DEVICE_KEY = 'matcharr:device'
const QUALITY_KEY = 'matcharr:quality'

interface StoredCreds {
  code: string
  token: string
}

export type Finale = 'match' | 'goal' | 'deck' | null
export interface Notice {
  kind: 'cancelled' | 'removed'
  host?: string
}

export const state = ref<PublicState | null>(null)
export const you = ref<You | null>(null)
export const deck = ref<Movie[]>([])
export const cursor = ref(0)
export const connected = ref(false)
export const busy = ref(false)
export const error = ref('')
export const deckLoading = ref(false)

export const matchToast = ref<Movie | null>(null)
export const finale = ref<Finale>(null)
export const notice = ref<Notice | null>(null)

export const invite = ref(inviteFromHash())

export const requestsEnabled = ref(false)
export const qualityProfiles = ref<QualityProfile[]>([])
export const qualityProfileId = ref(0)

export const inSession = computed(() => state.value !== null)
export const currentMovie = computed(() => deck.value[cursor.value] ?? null)
// An empty deck on an active session means it is still on the wire, not that
// the swiping is over: without this the end screen flashes on slow links.
export const deckPending = computed(() => deckLoading.value || deck.value.length === 0)
export const isHost = computed(() => you.value?.isHost ?? false)
export const canRequest = computed(() => requestsEnabled.value && isHost.value)
export const qualityProfileName = computed(
  () =>
    qualityProfiles.value.find((p) => p.id === qualityProfileId.value)?.name ??
    t('quality.default'),
)

export function hostName(): string {
  return state.value?.participants.find((p) => p.isHost)?.name ?? t('session.defaultHost')
}

let creds: StoredCreds | null = loadCreds()
let requestSetup: Promise<void> | null = null
let ws: WebSocket | null = null
let reconnectDelay = 1000
let reconnectTimer: number | undefined
let closedByUs = false

const detailsCache = new Map<number, Movie>()

function loadCreds(): StoredCreds | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as StoredCreds) : null
  } catch {
    return null
  }
}

export function savedProfile(): { name: string; avatar: string } {
  try {
    const raw = localStorage.getItem(NAME_KEY)
    if (raw) return JSON.parse(raw)
  } catch {
    /* ignore a corrupted localStorage */
  }
  return { name: '', avatar: '' }
}

export function updateProfile(name: string, avatar: string): void {
  localStorage.setItem(NAME_KEY, JSON.stringify({ name, avatar }))
}

// randomUUID is secure-context only; LAN clients run plain HTTP,
// so fall back to getRandomValues.
function deviceId(): string {
  let id = localStorage.getItem(DEVICE_KEY)
  if (!id) {
    if (typeof crypto.randomUUID === 'function') {
      id = crypto.randomUUID()
    } else {
      const bytes = new Uint8Array(16)
      crypto.getRandomValues(bytes)
      id = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('')
    }
    localStorage.setItem(DEVICE_KEY, id)
  }
  return id
}

function setCreds(next: StoredCreds | null) {
  creds = next
  if (next) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  } else {
    localStorage.removeItem(STORAGE_KEY)
  }
}

// The hash mirrors the session lifecycle via replaceState, which fires
// no hashchange and adds no history entries. The state object carries the
// back-navigation depth, so it has to survive the rewrite.
function setHash(code: string | null) {
  const base = location.pathname + location.search
  history.replaceState(history.state, '', code ? `${base}#/s/${code}` : base)
}

function inviteFromHash(): string {
  return location.hash.match(/^#\/s\/([A-Za-z0-9]+)/)?.[1]?.toUpperCase() ?? ''
}

export function clearSessionLink(): void {
  setHash(null)
}

// A pasted invite lands as a history entry the app has never seen, and it
// arrives on the same event as the back-stack repair, so the code is taken
// before the hash is rewritten. Entries the app has already visited keep the
// hash they were pushed with, and reviving that would rejoin a dead session.
export function settleSessionLink(fresh: boolean): void {
  const pasted = fresh ? inviteFromHash() : ''
  if (pasted && !creds) {
    invite.value = pasted
    return
  }
  setHash(creds?.code ?? null)
}

export async function resume(codeFromLink?: string): Promise<void> {
  if (!creds) return
  if (codeFromLink && codeFromLink !== creds.code) {
    reset()
    return
  }
  try {
    const personal = await api.fetchState(creds.code, creds.token)
    if (personal.state.status === 'finished' && personal.state.round === 0) {
      // finished at round 0 means cancelled before start
      reset()
      return
    }
    applyState(personal.state, personal.you)
    void loadRequestSetup()
    setHash(creds.code)
    if (personal.state.status === 'finished') {
        finale.value = null
    }
    if (personal.state.status !== 'lobby') {
      await loadDeck()
    }
    connect()
  } catch (e) {
    if (e instanceof api.ApiError && (e.status === 404 || e.status === 401)) {
      reset()
    } else {
      error.value = errorText(e)
    }
  }
}

export async function create(name: string, avatar: string, settings: Settings): Promise<void> {
  const c = await api.createSession(name, avatar, settings, deviceId())
  updateProfile(name, avatar)
  setCreds({ code: c.code, token: c.token })
  you.value = c.you
  await resume()
}

export async function join(code: string, name: string, avatar: string): Promise<void> {
  const c = await api.joinSession(code.toUpperCase().trim(), name, avatar, deviceId())
  updateProfile(name, avatar)
  setCreds({ code: c.code, token: c.token })
  you.value = c.you
  await resume()
}

export async function leave(): Promise<void> {
  if (creds && state.value && state.value.status !== 'finished' && !you.value?.isHost) {
    try {
      await api.leaveSession(creds.code, creds.token)
    } catch {
      /* leaving must always work locally, even when the server is gone */
    }
  }
  reset()
}

export function dismissNotice(): void {
  notice.value = null
}

function reset(keep?: Notice) {
  closedByUs = true
  ws?.close()
  ws = null
  window.clearTimeout(reconnectTimer)
  setCreds(null)
  setHash(null)
  state.value = null
  you.value = null
  deck.value = []
  cursor.value = 0
  connected.value = false
  matchToast.value = null
  finale.value = null
  notice.value = keep ?? null
  error.value = ''
  requestSetup = null
  requestsEnabled.value = false
  qualityProfiles.value = []
  qualityProfileId.value = 0
}

function applyState(next: PublicState, yourself?: You) {
  state.value = next
  if (yourself) {
    you.value = yourself
    cursor.value = yourself.cursor
  }
}

const DECK_ATTEMPTS = 3
const DECK_RETRY_DELAY = 2000

async function loadDeck(attempt = 1): Promise<void> {
  if (!creds) return
  deckLoading.value = true
  try {
    const { movies } = await api.fetchDeck(creds.code, creds.token)
    deck.value = movies
  } catch (e) {
    if (attempt < DECK_ATTEMPTS) {
      window.setTimeout(() => void loadDeck(attempt + 1), DECK_RETRY_DELAY)
      return
    }
    error.value = errorText(e)
  } finally {
    deckLoading.value = false
  }
}

function connect() {
  if (!creds || ws) return
  closedByUs = false
  const socket = new WebSocket(api.wsUrl(creds.code, creds.token))
  ws = socket

  socket.onopen = () => {
    connected.value = true
    reconnectDelay = 1000
  }
  socket.onmessage = (msg) => {
    try {
      handleEvent(JSON.parse(msg.data as string) as SessionEvent)
    } catch {
      /* ignore garbage on the socket */
    }
  }
  socket.onclose = () => {
    connected.value = false
    ws = null
    if (!closedByUs && creds) {
      reconnectTimer = window.setTimeout(() => {
        reconnectDelay = Math.min(reconnectDelay * 2, 10_000)
        void resume()
      }, reconnectDelay)
    }
  }
}

function handleEvent(ev: SessionEvent) {
  if (ev.state) {
    // finished at round 0 means the host cancelled the lobby
    if (ev.state.status === 'finished' && ev.state.round === 0) {
      const host = hostName()
      reset({ kind: 'cancelled', host })
      return
    }
    applyState(ev.state, ev.type === 'snapshot' ? ev.you : undefined)
  }
  switch (ev.type) {
    case 'session_started':
      void loadDeck()
      break
    case 'continue_result':
      if (ev.data?.extended) {
        void loadDeck()
      }
      break
    case 'participant_kicked':
      if (ev.data?.participantId === you.value?.id) {
        reset({ kind: 'removed', host: hostName() })
      }
      break
    case 'match_found': {
      const movie = (ev.data?.movie as Movie) ?? null
      const s = ev.state
      if (!movie || !s) break
      const goalHit =
        s.settings.matchMode === 'count' && s.matches.length >= (s.settings.matchTarget ?? 0)
      if (s.settings.matchMode !== 'first' && !goalHit) {
        matchToast.value = movie
      }
      break
    }
    case 'session_finished': {
      const s = ev.state
      if (!s) break
      matchToast.value = null
      finale.value = pickFinale(s)
      break
    }
  }
}

function pickFinale(s: PublicState): Finale {
  if (s.settings.matchMode === 'first' && s.matches.length > 0) return 'match'
  if (
    s.settings.matchMode === 'count' &&
    s.matches.length >= (s.settings.matchTarget ?? Infinity)
  ) {
    return 'goal'
  }
  if (s.settings.matchMode === 'all' && s.allDone) return 'deck'
  return null
}

export function dismissFinale(): void {
  finale.value = null
}

export function errorText(e: unknown): string {
  return e instanceof api.ApiError ? tReason(e.reason) : t('error.generic')
}

async function action(fn: (code: string, token: string) => Promise<unknown>): Promise<boolean> {
  if (!creds) return false
  busy.value = true
  error.value = ''
  try {
    await fn(creds.code, creds.token)
    return true
  } catch (e) {
    error.value = errorText(e)
    return false
  } finally {
    busy.value = false
  }
}

export function start(): Promise<boolean> {
  return action(api.startSession)
}

export async function swipe(verdict: Verdict): Promise<void> {
  const movie = currentMovie.value
  if (!creds || !movie) return
  cursor.value++
  try {
    await api.swipeMovie(creds.code, creds.token, movie.id, verdict)
  } catch (e) {
      error.value = errorText(e)
    await resync()
  }
}

export async function undo(): Promise<void> {
  if (!creds || cursor.value === 0) return
  try {
    await api.undoSwipe(creds.code, creds.token)
    cursor.value--
  } catch (e) {
    error.value = errorText(e)
  }
}

export function finish(): Promise<boolean> {
  return action(api.finishSession)
}

export function propose(limit: number): Promise<boolean> {
  return action((c, t) => api.proposeContinue(c, t, limit))
}

export function voteContinue(accept: boolean): Promise<boolean> {
  return action((c, t) => api.voteContinue(c, t, accept))
}

export function cancelProposal(): Promise<boolean> {
  return action(api.cancelContinue)
}

export function kick(participantId: string): Promise<boolean> {
  return action((c, t) => api.kickParticipant(c, t, participantId))
}

// The flag is public, the profiles are host-only: guests would get a 403,
// so they keep an empty list and no request controls.
function loadRequestSetup(): Promise<void> {
  requestSetup ??= readRequestSetup()
  return requestSetup
}

async function readRequestSetup(): Promise<void> {
  try {
    requestsEnabled.value = (await api.fetchConfig()).requestsEnabled
  } catch {
    requestsEnabled.value = false
  }
  if (!creds || !canRequest.value) return
  try {
    const { profiles, defaultId } = await api.fetchQualityProfiles(creds.code, creds.token)
    qualityProfiles.value = profiles
    const saved = Number(localStorage.getItem(QUALITY_KEY))
    qualityProfileId.value = profiles.some((p) => p.id === saved) ? saved : defaultId
  } catch {
    /* the backend falls back to the default quality profile */
  }
}

export function selectQualityProfile(id: number): void {
  qualityProfileId.value = id
  localStorage.setItem(QUALITY_KEY, String(id))
}

// Sent requests live in the session state, so a reload (or another
// participant's screen) still shows them as sent.
export function isRequested(movieId: number): boolean {
  return state.value?.requested?.includes(movieId) ?? false
}

export async function requestMovie(movieId: number): Promise<boolean> {
  const ok = await action((c, t) => api.requestMovie(c, t, movieId, qualityProfileId.value))
  if (ok) await resync()
  return ok
}

let genresCache: Genre[] | null = null

export async function genres(): Promise<Genre[]> {
  if (genresCache) return genresCache
  const list = await api.fetchGenres()
  genresCache = list
  return list
}

const PRELOAD_AHEAD = 6
const preloadedPosters = new Set<string>()

watch([deck, cursor], () => {
  const end = Math.min(cursor.value + PRELOAD_AHEAD, deck.value.length)
  for (let i = cursor.value; i < end; i++) {
    const path = deck.value[i]?.posterPath
    if (!path) continue
    const url = api.imageUrl(path, api.POSTER_FULL)
    if (preloadedPosters.has(url)) continue
    preloadedPosters.add(url)
    new Image().src = url
  }
})

export async function movieDetails(id: number): Promise<Movie | null> {
  const cached = detailsCache.get(id)
  if (cached) return cached
  if (!creds) return null
  try {
    const movie = await api.fetchMovieDetails(creds.code, creds.token, id)
    detailsCache.set(id, movie)
    return movie
  } catch {
    return null
  }
}

async function resync() {
  if (!creds) return
  try {
    const personal = await api.fetchState(creds.code, creds.token)
    applyState(personal.state, personal.you)
  } catch {
    /* the next WS snapshot will fix it */
  }
}
