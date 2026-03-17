'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useState } from 'react'
import { tAdmin } from '@/lib/i18n/admin'
import { useLocalePreference } from '@/lib/ui/preferences'

function isExactActive(pathname: string, href: string): boolean {
  return pathname === href
}

function NavIcon({ name }: { name: string }) {
  const common = {
    fill: 'none',
    viewBox: '0 0 24 24',
    strokeWidth: 1.8,
    stroke: 'currentColor',
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true,
  }

  switch (name) {
    case 'info':
      return <svg {...common}><circle cx="12" cy="12" r="9" /><path d="M12 10v6" /><path d="M12 7h.01" /></svg>
    case 'template':
      return <svg {...common}><rect x="5" y="4" width="14" height="16" rx="2" /><path d="M9 8h6M9 12h6M9 16h4" /></svg>
    case 'doc':
      return <svg {...common}><path d="M7 3h7l4 4v14H7z" /><path d="M14 3v5h5" /></svg>
    case 'dev':
      return <svg {...common}><rect x="4" y="4" width="16" height="16" rx="4" /><path d="M9 9h6v6H9z" /></svg>
    case 'crud':
      return <svg {...common}><path d="M8 16l8-8M10 7h7v7" /><path d="M8 8h.01M16 16h.01" /></svg>
    case 'cap':
      return <svg {...common}><path d="M12 3v4M12 17v4M3 12h4M17 12h4M5.6 5.6l2.8 2.8M15.6 15.6l2.8 2.8M18.4 5.6l-2.8 2.8M8.4 15.6l-2.8 2.8" /></svg>
    case 'life':
      return <svg {...common}><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></svg>
    case 'test':
      return <svg {...common}><path d="M9 3h6M10 3v4l-5 8a3 3 0 0 0 2.6 4.5h8.8A3 3 0 0 0 19 15l-5-8V3" /></svg>
    case 'overview':
      return <svg {...common}><circle cx="12" cy="12" r="9" /><path d="M12 12l5-2" /><path d="M12 12v6" /></svg>
    case 'members':
      return <svg {...common}><circle cx="9" cy="8" r="3" /><path d="M4 18a5 5 0 0 1 10 0" /><circle cx="17" cy="9" r="2" /><path d="M15 18h5" /></svg>
    case 'roles':
      return <svg {...common}><circle cx="9" cy="12" r="2" /><path d="M11 12h9" /><path d="M16 9l4 3-4 3" /><path d="M3 12h4" /></svg>
    case 'settings':
      return <svg {...common}><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1 1 0 0 0 .2 1.1l.1.1a1 1 0 0 1 0 1.4l-1 1a1 1 0 0 1-1.4 0l-.1-.1a1 1 0 0 0-1.1-.2 1 1 0 0 0-.6.9V20a1 1 0 0 1-1 1h-2a1 1 0 0 1-1-1v-.2a1 1 0 0 0-.6-.9 1 1 0 0 0-1.1.2l-.1.1a1 1 0 0 1-1.4 0l-1-1a1 1 0 0 1 0-1.4l.1-.1a1 1 0 0 0 .2-1.1 1 1 0 0 0-.9-.6H4a1 1 0 0 1-1-1v-2a1 1 0 0 1 1-1h.2a1 1 0 0 0 .9-.6 1 1 0 0 0-.2-1.1l-.1-.1a1 1 0 0 1 0-1.4l1-1a1 1 0 0 1 1.4 0l.1.1a1 1 0 0 0 1.1.2 1 1 0 0 0 .6-.9V4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v.2a1 1 0 0 0 .6.9 1 1 0 0 0 1.1-.2l.1-.1a1 1 0 0 1 1.4 0l1 1a1 1 0 0 1 0 1.4l-.1.1a1 1 0 0 0-.2 1.1 1 1 0 0 0 .9.6H20a1 1 0 0 1 1 1v2a1 1 0 0 1-1 1h-.2a1 1 0 0 0-.9.6z" /></svg>
    default:
      return <svg {...common}><circle cx="12" cy="12" r="8" /></svg>
  }
}

export default function AdminSidebar() {
  const pathname = usePathname()
  const locale = useLocalePreference()
  const [templateOpen, setTemplateOpen] = useState(true)
  const templateGroupActive = pathname === '/templates' || pathname.startsWith('/templates/')

  return (
    <aside className="px-shell-sidebar">
      <section className="px-shell-group">
        <div className="px-shell-group-items">
          <Link href="/intro" className={`px-shell-nav-item ${isExactActive(pathname, '/intro') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="info" /></span>
            <span>{tAdmin(locale, 'sidebar.intro')}</span>
          </Link>
        </div>
      </section>

      <section className="px-shell-group">
        <h3 className="px-shell-group-title">{tAdmin(locale, 'sidebar.templates')}</h3>
        <div className="px-shell-group-items">
          <button
            type="button"
            className={`px-shell-nav-item px-shell-nav-parent ${templateGroupActive ? 'is-active' : ''}`}
            onClick={() => setTemplateOpen((value) => !value)}
            aria-expanded={templateOpen}
          >
            <span className="px-shell-nav-icon"><NavIcon name="template" /></span>
            <span>{tAdmin(locale, 'sidebar.templates')}</span>
            <span className={`px-shell-nav-chevron ${templateOpen ? 'is-open' : ''}`} aria-hidden="true">⌄</span>
          </button>
          {templateOpen ? (
            <>
              <Link href="/templates" className={`px-shell-nav-item is-nested ${isExactActive(pathname, '/templates') ? 'is-active' : ''}`}>
                <span className="px-shell-nav-icon"><NavIcon name="doc" /></span>
                <span>{tAdmin(locale, 'sidebar.templates.overview')}</span>
              </Link>
              <Link href="/templates/develop" className={`px-shell-nav-item is-nested ${isExactActive(pathname, '/templates/develop') ? 'is-active' : ''}`}>
                <span className="px-shell-nav-icon"><NavIcon name="dev" /></span>
                <span>{tAdmin(locale, 'sidebar.templates.develop')}</span>
              </Link>
              <Link href="/templates/crud" className={`px-shell-nav-item is-nested ${isExactActive(pathname, '/templates/crud') ? 'is-active' : ''}`}>
                <span className="px-shell-nav-icon"><NavIcon name="crud" /></span>
                <span>{tAdmin(locale, 'sidebar.templates.crud')}</span>
              </Link>
            </>
          ) : null}
        </div>
      </section>

      <section className="px-shell-group">
        <h3 className="px-shell-group-title">{tAdmin(locale, 'sidebar.capability')}</h3>
        <div className="px-shell-group-items">
          <Link href="/capabilities/register" className={`px-shell-nav-item ${isExactActive(pathname, '/capabilities/register') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="cap" /></span>
            <span>{tAdmin(locale, 'sidebar.capability.register')}</span>
          </Link>
          <Link href="/capabilities/lifecycle" className={`px-shell-nav-item ${isExactActive(pathname, '/capabilities/lifecycle') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="life" /></span>
            <span>{tAdmin(locale, 'sidebar.capability.lifecycle')}</span>
          </Link>
          <Link
            href="/powerx/capability-lab?source=all"
            className={`px-shell-nav-item ${(isExactActive(pathname, '/tests/capability') || isExactActive(pathname, '/powerx/capability-lab')) ? 'is-active' : ''}`}
          >
            <span className="px-shell-nav-icon"><NavIcon name="test" /></span>
            <span>{tAdmin(locale, 'sidebar.capability.test')}</span>
          </Link>
        </div>
      </section>

      <section className="px-shell-group">
        <h3 className="px-shell-group-title">{tAdmin(locale, 'sidebar.iam')}</h3>
        <div className="px-shell-group-items">
          <Link href="/admin/iam/overview" className={`px-shell-nav-item ${isExactActive(pathname, '/admin/iam/overview') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="overview" /></span>
            <span>{tAdmin(locale, 'sidebar.iam.overview')}</span>
          </Link>
          <Link href="/admin/iam/members" className={`px-shell-nav-item ${isExactActive(pathname, '/admin/iam/members') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="members" /></span>
            <span>{tAdmin(locale, 'sidebar.iam.members')}</span>
          </Link>
          <Link href="/admin/iam/roles" className={`px-shell-nav-item ${isExactActive(pathname, '/admin/iam/roles') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="roles" /></span>
            <span>{tAdmin(locale, 'sidebar.iam.roles')}</span>
          </Link>
          <Link href="/admin/iam/settings" className={`px-shell-nav-item ${isExactActive(pathname, '/admin/iam/settings') ? 'is-active' : ''}`}>
            <span className="px-shell-nav-icon"><NavIcon name="settings" /></span>
            <span>{tAdmin(locale, 'sidebar.iam.settings')}</span>
          </Link>
        </div>
      </section>
    </aside>
  )
}
