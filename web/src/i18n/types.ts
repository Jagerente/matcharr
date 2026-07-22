import type en from './en'

export type Locale = 'en' | 'ru' | 'zh-Hans'

export interface PluralForms {
  one?: string
  few?: string
  many?: string
  other: string
}

export type MessageKey = keyof typeof en

export type Messages = {
  [K in MessageKey]: (typeof en)[K] extends string ? string : PluralForms
}

export type MessageParams = Record<string, string | number>
