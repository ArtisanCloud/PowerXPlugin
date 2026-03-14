import Link from 'next/link'

export default function IntroPage() {
  return (
    <main style={{ padding: 24, display: 'grid', gap: 16 }}>
      <h1 data-testid="intro-title" style={{ margin: 0 }}>管理首页</h1>
      <p style={{ margin: 0, color: '#475569' }}>此页面用于 Next 管理端迁移联调入口。</p>
      <div style={{ display: 'flex', gap: 12 }}>
        <Link data-testid="intro-link-templates-crud" href="/templates/crud">模板 CRUD</Link>
        <Link data-testid="intro-link-templates-develop" href="/templates/develop">模板开发指引</Link>
      </div>
    </main>
  )
}
