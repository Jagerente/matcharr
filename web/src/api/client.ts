

export interface Preset {
  id: string
  title: Record<string, string>
  description?: Record<string, string>
  icon?: string
  mediaType: string
}

export interface Movie {
  id: number
  title: string
  originalTitle?: string
  year?: number
  genres?: string[]
  rating?: number
  voteCount?: number
  overview?: string
  posterPath?: string
  backdropPath?: string
  runtime?: number
}

export type MatchMode = 'first' | 'count' | 'all'
export type SessionStatus = 'lobby' | 'active' | 'voting' | 'finished'
export type Verdict = 'like' | 'nope'

export interface Genre {
  id: number
  name: string
}

export interface Filters {
  genreIds?: number[]
  yearFrom?: number
  yearTo?: number
  ratingMin?: number
  votesMin?: number
}

export interface Settings {
  mediaType?: string
  presetId: string
  matchMode: MatchMode
  matchTarget?: number
  threshold?: number
  deckLimit: number
  filters?: Filters
}

export interface Participant {
  id: string
  name: string
  avatar?: string
  isHost: boolean
  online: boolean
  progress: number
  done: boolean
}

export interface Match {
  movie: Movie
  matchedAt: string
  votes?: Record<string, Verdict>
}

export interface ContinueVote {
  proposedLimit: number
  votes: Record<string, boolean>
}

export interface NearMatch {
  movie: Movie
  likes: number
  needed: number
  votes?: Record<string, Verdict>
}

export interface ParticipantStats {
  id: string
  name: string
  likes: number
  seen: number
}

export interface Results {
  matches: Match[]
  nearMatches: NearMatch[]
  stats: ParticipantStats[]
}

export interface PublicState {
  code: string
  status: SessionStatus
  round: number
  settings: Settings
  participants: Participant[]
  departed?: Participant[]
  deckSize: number
  sourceExhausted: boolean
  allDone: boolean
  nearMissCount: number
  matches: Match[]
  requested?: number[]
  continue?: ContinueVote
  results?: Results
}

export interface You {
  id: string
  isHost: boolean
  cursor: number
}

export interface PersonalState {
  state: PublicState
  you: You
}

export interface Credentials {
  code: string
  token: string
  you: You
}

export interface AppConfig {
  requestsEnabled: boolean
}

export interface QualityProfile {
  id: number
  name: string
}

export interface QualityProfiles {
  profiles: QualityProfile[]
  defaultId: number
}

export interface SessionEvent {
  type: string
  state?: PublicState
  you?: You
  data?: Record<string, unknown>
}


// The message is technical, meant for the console. User-facing copy comes from the reason.
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly reason: string,
  ) {
    super(message)
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, init)
  if (!response.ok) {
    const body = await response.json().catch(() => null)
    const message = body?.message ?? `HTTP ${response.status}`
    throw new ApiError(message, response.status, body?.reason ?? 'internal')
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json()
}

function withToken(token: string, body?: unknown): RequestInit {
  const init: RequestInit = {
    method: 'POST',
    headers: { 'X-Session-Token': token },
  }
  if (body !== undefined) {
    init.headers = { ...init.headers, 'Content-Type': 'application/json' }
    init.body = JSON.stringify(body)
  }
  return init
}

export function fetchConfig(): Promise<AppConfig> {
  return request('api/config')
}

export function fetchPresets(): Promise<Preset[]> {
  return request('api/presets')
}

export function fetchGenres(): Promise<Genre[]> {
  return request('api/genres')
}

export function fetchMovieDetails(code: string, token: string, id: number): Promise<Movie> {
  return request(`api/sessions/${code}/movies/${id}`, { headers: { 'X-Session-Token': token } })
}

// Only two variants exist so a poster is fetched and decoded once per session:
// FULL is what the swipe deck preloads, THUMB is what grids of many posters use.
export const POSTER_FULL = 'w500'
export const POSTER_THUMB = 'w185'

export function imageUrl(posterPath: string, size = POSTER_THUMB): string {
  return `api/image/${size}/${posterPath.replace(/^\//, '')}`
}


export function createSession(
  name: string,
  avatar: string,
  settings: Settings,
  deviceId: string,
): Promise<Credentials> {
  return request('api/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, avatar, settings, deviceId }),
  })
}

export function joinSession(
  code: string,
  name: string,
  avatar: string,
  deviceId: string,
): Promise<Credentials> {
  return request(`api/sessions/${code}/join`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, avatar, deviceId }),
  })
}

export function fetchState(code: string, token: string): Promise<PersonalState> {
  return request(`api/sessions/${code}`, { headers: { 'X-Session-Token': token } })
}

export function fetchDeck(code: string, token: string): Promise<{ movies: Movie[] }> {
  return request(`api/sessions/${code}/deck`, { headers: { 'X-Session-Token': token } })
}

export function startSession(code: string, token: string): Promise<unknown> {
  return request(`api/sessions/${code}/start`, withToken(token, {}))
}

export function swipeMovie(code: string, token: string, movieId: number, verdict: Verdict): Promise<unknown> {
  return request(`api/sessions/${code}/swipe`, withToken(token, { movieId, verdict }))
}

export function undoSwipe(code: string, token: string): Promise<unknown> {
  return request(`api/sessions/${code}/undo`, withToken(token, {}))
}

export function finishSession(code: string, token: string): Promise<unknown> {
  return request(`api/sessions/${code}/finish`, withToken(token, {}))
}

export function leaveSession(code: string, token: string): Promise<unknown> {
  return request(`api/sessions/${code}/leave`, withToken(token, {}))
}

export function proposeContinue(code: string, token: string, limit: number): Promise<unknown> {
  return request(`api/sessions/${code}/continue/propose`, withToken(token, { limit }))
}

export function voteContinue(code: string, token: string, accept: boolean): Promise<unknown> {
  return request(`api/sessions/${code}/continue/vote`, withToken(token, { accept }))
}

export function cancelContinue(code: string, token: string): Promise<unknown> {
  return request(`api/sessions/${code}/continue/cancel`, withToken(token, {}))
}

export function kickParticipant(code: string, token: string, participantId: string): Promise<unknown> {
  return request(`api/sessions/${code}/participants/${participantId}`, {
    method: 'DELETE',
    headers: { 'X-Session-Token': token },
  })
}

export function fetchQualityProfiles(code: string, token: string): Promise<QualityProfiles> {
  return request(`api/sessions/${code}/request/profiles`, { headers: { 'X-Session-Token': token } })
}

export function requestMovie(
  code: string,
  token: string,
  movieId: number,
  profileId: number,
): Promise<unknown> {
  return request(`api/sessions/${code}/request`, withToken(token, { movieId, profileId }))
}


export function basePath(): string {
  return location.pathname.replace(/[^/]*$/, '')
}

export function shareLink(code: string): string {
  return `${location.origin}${basePath()}#/s/${code}`
}

export function wsUrl(code: string, token: string): string {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${location.host}${basePath()}api/sessions/${code}/ws?token=${encodeURIComponent(token)}`
}
