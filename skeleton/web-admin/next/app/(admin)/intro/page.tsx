import Link from 'next/link'

export default function IntroPage() {
  return (
    <main className="px-admin-page">
      <section className="px-admin-shell">
        <article className="px-admin-intro-head">
          <h1 data-testid="intro-title" className="px-admin-title">PowerX Plugin 管理台</h1>
          <p className="px-admin-subtitle">这里是与 Nuxt Admin 对齐的 Next 迁移入口，包含模板、能力注册与 IAM 联调工作流。</p>
        </article>

        <section className="px-admin-intro-grid">
          <article className="px-admin-card">
            <h3 className="px-admin-card-title">快速启动</h3>
            <p className="px-admin-card-text">通过模板和能力页面快速验证插件脚手架、接口约定与前后端联通状态。</p>
          </article>
          <article className="px-admin-card">
            <h3 className="px-admin-card-title">可复用流程</h3>
            <p className="px-admin-card-text">保持与 Nuxt 相同的导航语义和页面层级，减少迁移期间的学习与维护成本。</p>
          </article>
          <article className="px-admin-card">
            <h3 className="px-admin-card-title">接口联调</h3>
            <p className="px-admin-card-text">登录态、鉴权模式和 API 代理链路均可直接在此页面入口下验证。</p>
          </article>
        </section>

        <section>
          <h2 className="px-admin-section-title">下一步</h2>
          <div className="px-admin-next-grid">
            <article className="px-admin-card">
              <h3 className="px-admin-card-title">前往模板 CRUD</h3>
              <p className="px-admin-card-text">验证资源页面、表格和权限可见性是否符合预期。</p>
              <div className="px-admin-toolbar">
                <Link data-testid="intro-link-templates-crud" className="px-btn" href="/templates/crud">查看 CRUD</Link>
              </div>
            </article>
            <article className="px-admin-card">
              <h3 className="px-admin-card-title">查看开发指引</h3>
              <p className="px-admin-card-text">按模板开发文档继续扩展页面与后端能力。</p>
              <div className="px-admin-toolbar">
                <Link data-testid="intro-link-templates-develop" className="px-btn-ghost" href="/templates/develop">查看指引</Link>
              </div>
            </article>
          </div>
        </section>
      </section>
    </main>
  )
}
