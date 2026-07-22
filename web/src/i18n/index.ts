import { computed, ref } from 'vue'
import en from './en'
import ru from './ru'
import zhHans from './zh-Hans'
import type { Locale, MessageKey, MessageParams, Messages, PluralForms } from './types'

export type { Locale, MessageKey }

export interface LocaleOption {
  id: Locale
  name: string
  short: string
}

export const LOCALES: readonly LocaleOption[] = [
  { id: 'en', name: 'English', short: 'EN' },
  { id: 'ru', name: 'Русский', short: 'RU' },
  { id: 'zh-Hans', name: '简体中文', short: '中文' },
]

const STORAGE_KEY = 'matcharr:locale'

const BUNDLES: Record<Locale, Messages> = { en, ru, 'zh-Hans': zhHans }
const TAGS: Record<Locale, string> = { en: 'en', ru: 'ru', 'zh-Hans': 'zh-Hans' }

const locale = ref<Locale>(initial())

export const currentLocale = computed(
  () => LOCALES.find((l) => l.id === locale.value) ?? LOCALES[0]!,
)

export function setLocale(next: Locale): void {
  if (!(next in BUNDLES)) return
  locale.value = next
  localStorage.setItem(STORAGE_KEY, next)
  syncDocumentLang()
}

export function syncDocumentLang(): void {
  document.documentElement.lang = TAGS[locale.value]
}

export function t(key: MessageKey, params?: MessageParams): string {
  const message = BUNDLES[locale.value][key] ?? en[key]
  const template =
    typeof message === 'string' ? message : plural(message, Number(params?.count ?? 0))
  return fill(template, params)
}

export function tReason(reason: string): string {
  const key = `error.${reason}` as MessageKey
  return key in en ? t(key) : t('error.generic')
}

export function pickLocalized(map: Record<string, string> | undefined): string {
  if (!map) return ''
  return map[locale.value] ?? map.en ?? Object.values(map)[0] ?? ''
}

function initial(): Locale {
  const manual = localStorage.getItem(STORAGE_KEY)
  if (manual && manual in BUNDLES) return manual as Locale
  for (const tag of navigator.languages?.length ? navigator.languages : [navigator.language]) {
    const detected = fromTag(tag)
    if (detected) return detected
  }
  return 'en'
}

// Traditional Chinese falls through to the next preferred language:
// only the Simplified bundle ships.
function fromTag(tag: string): Locale | null {
  const lower = tag.toLowerCase()
  if (lower.startsWith('ru')) return 'ru'
  if (lower.startsWith('en')) return 'en'
  if (lower.startsWith('zh') && !/hant|-(tw|hk|mo)$/.test(lower)) return 'zh-Hans'
  return null
}

const rules = new Map<Locale, Intl.PluralRules>()

function plural(forms: PluralForms, count: number): string {
  switch (rulesFor(locale.value).select(count)) {
    case 'one':
      return forms.one ?? forms.other
    case 'few':
      return forms.few ?? forms.other
    case 'many':
      return forms.many ?? forms.other
    default:
      return forms.other
  }
}

function rulesFor(loc: Locale): Intl.PluralRules {
  let cached = rules.get(loc)
  if (!cached) {
    cached = new Intl.PluralRules(TAGS[loc])
    rules.set(loc, cached)
  }
  return cached
}

function fill(template: string, params?: MessageParams): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (placeholder, name: string) =>
    name in params ? String(params[name]) : placeholder,
  )
}
