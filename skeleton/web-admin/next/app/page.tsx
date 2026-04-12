import Link from 'next/link'

const heroCards = [
  {
    title: '快速入门',
    body: '通过模板管理示例了解插件配置、菜单与权限声明方式。',
  },
  {
    title: '可复用模板',
    body: '将常用片段抽象为模板，在多个页面中复用，保持体验一致。',
  },
  {
    title: 'API 演示',
    body: '后端提供最小实现的模板 API，可直接对接 PowerX 平台。',
  },
]

const nextSteps = [
  {
    title: '探索模板 CRUD',
    body: '在“模板 CRUD”页面体验完整的模板列表、创建、编辑和删除流程。',
    to: '/templates/crud',
    secondary: '/templates',
    secondaryLabel: '查看模板总览',
  },
  {
    title: '了解开发指引',
    body: '“开发指南”页面说明目录结构、依赖注入与接口注册等关键步骤。',
    to: '/templates/develop',
    secondary: '/intro',
    secondaryLabel: '查看介绍页',
  },
]

export default function HomePage() {
  return (
    <main className="px-landing-page">
      <section className="px-landing-hero">
        <div className="px-landing-container px-landing-hero-grid">
          <div className="px-landing-left">
            <span className="px-landing-pill">PowerX 基础插件</span>
            <h1 className="px-landing-title">PowerX 基础插件</h1>
            <p className="px-landing-desc">
              欢迎使用 PowerX 插件脚手架示例页面，快速了解模板 CRUD、菜单导航与多语言体验。
            </p>

            <div className="px-landing-actions">
              <Link className="px-landing-btn-primary" href="/intro">查看介绍</Link>
              <Link className="px-landing-btn-ghost" href="/users/login">体验登录</Link>
            </div>

            <div className="px-landing-stats">
              <div className="px-landing-stat">
                <p>前端壳层</p>
                <strong>Nuxt 4.2</strong>
              </div>
              <div className="px-landing-stat">
                <p>后端服务</p>
                <strong>Go 1.24</strong>
              </div>
            </div>
          </div>

          <div className="px-landing-right">
            <article className="px-landing-feature-card">
              <header>
                <h2>Base 模板插件</h2>
                <p>了解 PowerX 基础模板插件，快速搭建可复用的业务模块。</p>
              </header>

              <div className="px-landing-feature-list">
                {heroCards.map((card) => (
                  <div key={card.title} className="px-landing-feature-item">
                    <h3>{card.title}</h3>
                    <p>{card.body}</p>
                  </div>
                ))}
              </div>
            </article>
          </div>
        </div>
      </section>

      <section className="px-landing-next">
        <div className="px-landing-container">
          <h2 className="px-landing-next-title">下一步</h2>
          <p className="px-landing-next-desc">
            欢迎使用 PowerX 插件脚手架示例页面，快速了解模板 CRUD、菜单导航与多语言体验。
          </p>

          <div className="px-landing-next-grid">
            {nextSteps.map((item) => (
              <article key={item.title} className="px-landing-next-card">
                <h3>{item.title}</h3>
                <p>{item.body}</p>
                <div className="px-landing-next-actions">
                  <Link className="px-landing-btn-primary" href={item.to}>进入</Link>
                  <Link className="px-landing-btn-ghost" href={item.secondary}>{item.secondaryLabel}</Link>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>
    </main>
  )
}
