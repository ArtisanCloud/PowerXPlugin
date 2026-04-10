'use client'

import { useEffect, useState } from 'react'

export type AdminLocale = 'zh-CN' | 'en'
export type AdminTheme = 'light' | 'dark'

export const LOCALE_STORAGE_KEY = 'i18n_redirected'
export const THEME_STORAGE_KEY = 'px_theme'

const LOCALE_EVENT = 'px:locale-change'
const THEME_EVENT = 'px:theme-change'

export function readLocale(): AdminLocale {
  if (typeof window === 'undefined') return 'zh-CN'
  const value = (localStorage.getItem(LOCALE_STORAGE_KEY) || '').trim()
  return value === 'en' ? 'en' : 'zh-CN'
}

export function readTheme(): AdminTheme {
  if (typeof window === 'undefined') return 'light'
  const value = (localStorage.getItem(THEME_STORAGE_KEY) || '').trim().toLowerCase()
  return value === 'dark' ? 'dark' : 'light'
}

export function setLocalePreference(locale: AdminLocale) {
  if (typeof window === 'undefined') return
  localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  document.documentElement.lang = locale
  window.dispatchEvent(new CustomEvent(LOCALE_EVENT, { detail: locale }))
}

export function setThemePreference(theme: AdminTheme) {
  if (typeof window === 'undefined') return
  localStorage.setItem(THEME_STORAGE_KEY, theme)
  document.documentElement.setAttribute('data-theme', theme)
  document.documentElement.classList.toggle('dark', theme === 'dark')
  window.dispatchEvent(new CustomEvent(THEME_EVENT, { detail: theme }))
}

export function useLocalePreference() {
  const [locale, setLocale] = useState<AdminLocale>('zh-CN')

  useEffect(() => {
    setLocale(readLocale())
    const onLocaleChanged = (event: Event) => {
      const custom = event as CustomEvent<AdminLocale>
      setLocale(custom.detail === 'en' ? 'en' : 'zh-CN')
    }
    const onStorage = (event: StorageEvent) => {
      if (event.key === LOCALE_STORAGE_KEY) {
        setLocale(readLocale())
      }
    }
    window.addEventListener(LOCALE_EVENT, onLocaleChanged as EventListener)
    window.addEventListener('storage', onStorage)
    return () => {
      window.removeEventListener(LOCALE_EVENT, onLocaleChanged as EventListener)
      window.removeEventListener('storage', onStorage)
    }
  }, [])

  return locale
}

export function useThemePreference() {
  const [theme, setTheme] = useState<AdminTheme>('light')

  useEffect(() => {
    setTheme(readTheme())
    const onThemeChanged = (event: Event) => {
      const custom = event as CustomEvent<AdminTheme>
      setTheme(custom.detail === 'dark' ? 'dark' : 'light')
    }
    const onStorage = (event: StorageEvent) => {
      if (event.key === THEME_STORAGE_KEY) {
        setTheme(readTheme())
      }
    }
    window.addEventListener(THEME_EVENT, onThemeChanged as EventListener)
    window.addEventListener('storage', onStorage)
    return () => {
      window.removeEventListener(THEME_EVENT, onThemeChanged as EventListener)
      window.removeEventListener('storage', onStorage)
    }
  }, [])

  return theme
}
