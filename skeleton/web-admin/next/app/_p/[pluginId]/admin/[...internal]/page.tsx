import Link from 'next/link'

type HostProxyPageProps = {
  params: {
    pluginId: string
    internal?: string[]
  }
}

export default function HostProxyPage({ params }: HostProxyPageProps) {
  const segments = Array.isArray(params.internal) ? params.internal : []
  const internalPath = `/${segments.join('/')}`
  const target = internalPath === '/' ? '/intro' : internalPath

  return (
    <main className="px-admin-page" data-testid="host-proxy-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <h1 className="px-admin-title">宿主路径透传页</h1>
          <p className="px-admin-subtitle">用于 host 模式路径代理验证。</p>
          <p>pluginId: <strong data-testid="host-plugin-id">{params.pluginId}</strong></p>
          <p>internal: <strong data-testid="host-internal-path">{internalPath}</strong></p>
          <Link data-testid="host-target-link" className="px-btn" href={target} style={{ display: 'inline-flex', alignItems: 'center' }}>
            跳转到插件内部路径
          </Link>
        </article>
      </section>
    </main>
  )
}
