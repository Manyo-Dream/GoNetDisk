import { ref, reactive } from 'vue'
import en from './en.js'
import zhCN from './zh-CN.js'

const messages = { en, 'zh-CN': zhCN }

const LOCALE_KEY = 'gonetdisk_locale'

function detectLocale() {
  const stored = localStorage.getItem(LOCALE_KEY)
  if (stored && messages[stored]) return stored
  const nav = navigator.language || ''
  if (nav.startsWith('zh')) return 'zh-CN'
  return 'en'
}

const locale = ref(detectLocale())

function resolve(key) {
  const keys = key.split('.')
  let value = messages[locale.value]
  for (const k of keys) {
    if (value == null) break
    value = value[k]
  }
  return value
}

function t(key, params = {}) {
  let value = resolve(key)
  if (value === undefined || value === null) {
    console.warn(`[i18n] missing key: ${key}`)
    return key
  }
  if (typeof value === 'string') {
    return value.replace(/\{(\w+)\}/g, (_, k) =>
      params[k] !== undefined ? String(params[k]) : `{${k}}`
    )
  }
  return String(value)
}

let _i18n = null

export function createI18n() {
  if (_i18n) return _i18n
  _i18n = reactive({ locale, t })
  return _i18n
}

export function useI18n() {
  return createI18n()
}

export function setLocale(lang) {
  if (messages[lang]) {
    locale.value = lang
    localStorage.setItem(LOCALE_KEY, lang)
  }
}

export function toggleLocale() {
  setLocale(locale.value === 'zh-CN' ? 'en' : 'zh-CN')
}

export default {
  install(app) {
    const i18n = createI18n()
    app.config.globalProperties.$t = i18n.t
    app.config.globalProperties.$locale = i18n.locale
    app.provide('i18n', i18n)
  },
}
