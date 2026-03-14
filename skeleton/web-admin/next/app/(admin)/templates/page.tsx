import Link from 'next/link'

export default function TemplatesPage() {
  return (
    <main style={{ padding: 24, display: 'grid', gap: 12 }}>
      <h1 data-testid="templates-overview-title" style={{ margin: 0 }}>模板管理</h1>
      <p style={{ color: '#475569', margin: 0 }}>模板列表、CRUD 与开发指引页面。</p>
      <div style={{ display: 'flex', gap: 12 }}>
        <Link data-testid="templates-to-crud" href="/templates/crud">进入 CRUD</Link>
        <Link data-testid="templates-to-develop" href="/templates/develop">进入 Develop</Link>
      </div>
    </main>
  )
}
