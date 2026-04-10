'use client'

import { useEffect, useRef, useState } from 'react'
import Image from 'next/image'
import Link from 'next/link'
import { tAdmin } from '@/lib/i18n/admin'
import {
  readLocale,
  readTheme,
  setLocalePreference,
  setThemePreference,
  useLocalePreference,
  useThemePreference,
  type AdminLocale,
  type AdminTheme,
} from '@/lib/ui/preferences'

function BellIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M15 17h5l-1.4-1.4A2 2 0 0 1 18 14.2V11a6 6 0 1 0-12 0v3.2a2 2 0 0 1-.6 1.4L4 17h5" />
      <path d="M10 19a2 2 0 0 0 4 0" />
    </svg>
  )
}

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
    </svg>
  )
}

function ChevronDownIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m6 9 6 6 6-6" />
    </svg>
  )
}

export default function AdminNavbar() {
  const [theme, setTheme] = useState<AdminTheme>('light')
  const [locale, setLocale] = useState<AdminLocale>('zh-CN')
  const [notifyOpen, setNotifyOpen] = useState(false)
  const [themeOpen, setThemeOpen] = useState(false)
  const [localeOpen, setLocaleOpen] = useState(false)
  const [userOpen, setUserOpen] = useState(false)
  const actionsRef = useRef<HTMLDivElement | null>(null)
  const syncedLocale = useLocalePreference()
  const syncedTheme = useThemePreference()

  useEffect(() => {
    const nextTheme = readTheme()
    setTheme(nextTheme)
    setThemePreference(nextTheme)

    const nextLocale = readLocale()
    setLocale(nextLocale)
    setLocalePreference(nextLocale)
  }, [])

  useEffect(() => {
    setLocale(syncedLocale)
  }, [syncedLocale])

  useEffect(() => {
    setTheme(syncedTheme)
  }, [syncedTheme])

  useEffect(() => {
    const onDocClick = (event: MouseEvent) => {
      const target = event.target as Node
      if (!actionsRef.current?.contains(target)) {
        setNotifyOpen(false)
        setThemeOpen(false)
        setLocaleOpen(false)
        setUserOpen(false)
      }
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [])

  const handleThemeChange = (nextTheme: 'light' | 'dark') => {
    setTheme(nextTheme)
    setThemePreference(nextTheme)
    setThemeOpen(false)
  }

  const handleLocaleChange = (nextLocale: 'zh-CN' | 'en') => {
    setLocale(nextLocale)
    setLocalePreference(nextLocale)
    setLocaleOpen(false)
  }

  return (
    <header className="px-shell-navbar">
      <div className="px-shell-brand">
        <Image
          src="/images/logo.png"
          alt="PowerX Plugin Logo"
          className="px-shell-logo"
          width={32}
          height={32}
          priority
        />
        <div>
          <div className="px-shell-title">PowerX 基础插件 <span className="px-shell-version">v0.1.0</span></div>
        </div>
      </div>

      <div className="px-shell-center">
        <span className="px-shell-badge">{tAdmin(locale, 'navbar.iam')}</span>
        <span className="px-shell-center-text">{tAdmin(locale, 'navbar.iamDesc')}</span>
      </div>

      <div className="px-shell-actions px-shell-actions-panel" ref={actionsRef}>
        <div className="px-shell-dropdown-wrap">
          <button className="px-shell-icon-btn" type="button" aria-label="通知" onClick={() => setNotifyOpen((value) => !value)}>
            <BellIcon />
          </button>
          {notifyOpen ? (
            <div className="px-shell-dropdown-menu px-shell-notify-menu">
              <p>{tAdmin(locale, 'navbar.notify.empty')}</p>
            </div>
          ) : null}
        </div>

        <div className="px-shell-dropdown-wrap">
          <button className="px-shell-icon-btn" type="button" aria-label="主题切换" onClick={() => setThemeOpen((value) => !value)}>
            {theme === 'dark' ? <MoonIcon /> : <SunIcon />}
          </button>
          {themeOpen ? (
            <div className="px-shell-dropdown-menu">
              <button type="button" className={`px-shell-dropdown-item ${theme === 'light' ? 'is-active' : ''}`} onClick={() => handleThemeChange('light')}>{tAdmin(locale, 'navbar.theme.light')}</button>
              <button type="button" className={`px-shell-dropdown-item ${theme === 'dark' ? 'is-active' : ''}`} onClick={() => handleThemeChange('dark')}>{tAdmin(locale, 'navbar.theme.dark')}</button>
            </div>
          ) : null}
        </div>

        <div className="px-shell-dropdown-wrap">
          <button className="px-shell-lang-btn" type="button" aria-label="语言切换" onClick={() => setLocaleOpen((value) => !value)}>
            <span>{locale === 'en' ? tAdmin(locale, 'navbar.locale.en') : tAdmin(locale, 'navbar.locale.zh')}</span>
            <span className="px-shell-caret"><ChevronDownIcon /></span>
          </button>
          {localeOpen ? (
            <div className="px-shell-dropdown-menu">
              <button type="button" className={`px-shell-dropdown-item ${locale === 'zh-CN' ? 'is-active' : ''}`} onClick={() => handleLocaleChange('zh-CN')}>{tAdmin(locale, 'navbar.locale.zh')}</button>
              <button type="button" className={`px-shell-dropdown-item ${locale === 'en' ? 'is-active' : ''}`} onClick={() => handleLocaleChange('en')}>{tAdmin(locale, 'navbar.locale.en')}</button>
            </div>
          ) : null}
        </div>

        <div className="px-shell-dropdown-wrap">
          <button className="px-shell-avatar" type="button" aria-label="用户菜单" onClick={() => setUserOpen((value) => !value)}>MH</button>
          {userOpen ? (
            <div className="px-shell-dropdown-menu px-shell-user-menu">
              <Link href="/users/login" className="px-shell-dropdown-item">{tAdmin(locale, 'navbar.user.logout')}</Link>
            </div>
          ) : null}
        </div>
      </div>
    </header>
  )
}
