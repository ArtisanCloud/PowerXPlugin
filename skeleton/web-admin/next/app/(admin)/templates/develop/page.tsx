export default function TemplatesDevelopPage() {
  return (
    <main className="px-admin-page">
      <section className="px-admin-shell">
        <article className="px-admin-intro-head">
          <h1 data-testid="templates-develop-title" className="px-admin-title">开发介绍</h1>
          <p className="px-admin-subtitle">了解模板模块在后端与前端的关键实现步骤。</p>
        </article>

        <article className="px-dev-card">
          <header className="px-dev-card-header">
            <span className="px-dev-card-icon" aria-hidden="true">▣</span>
            <h2 className="px-dev-card-title">核心目录</h2>
          </header>
          <ul data-testid="templates-develop-structure" className="px-dev-list">
            <li>backend/internal/entity/models/template/template.go</li>
            <li>backend/internal/services/admin/templates/template_service.go</li>
            <li>backend/internal/transport/http/admin/templates/*</li>
            <li>web-admin/app/pages/templates</li>
          </ul>
        </article>

        <article className="px-dev-card">
          <header className="px-dev-card-header">
            <span className="px-dev-card-icon" aria-hidden="true">☰</span>
            <h2 className="px-dev-card-title">实施步骤</h2>
          </header>
          <ol className="px-dev-steps">
            <li>定义模板的 GORM 模型与迁移。</li>
            <li>实现仓储与服务层封装业务逻辑。</li>
            <li>在 /api/v1/templates 下注册受 RBAC 保护的 HTTP 接口。</li>
            <li>将 Nuxt 页面接入 API，并优化用户体验。</li>
          </ol>
        </article>
      </section>
    </main>
  )
}
