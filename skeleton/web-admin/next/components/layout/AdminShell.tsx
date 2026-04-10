import type { ReactNode } from 'react'
import AdminNavbar from './AdminNavbar'
import AdminSidebar from './AdminSidebar'

export default function AdminShell({ children }: { children: ReactNode }) {
  return (
    <div className="px-shell-root">
      <AdminNavbar />
      <div className="px-shell-main">
        <AdminSidebar />
        <main className="px-shell-content">{children}</main>
      </div>
    </div>
  )
}
