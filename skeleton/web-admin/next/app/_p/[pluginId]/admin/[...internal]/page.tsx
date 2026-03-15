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
    <main style={{ padding: 24 }} data-testid="host-proxy-page">
      <h1>宿主路径透传页</h1>
      <p>pluginId: <strong data-testid="host-plugin-id">{params.pluginId}</strong></p>
      <p>internal: <strong data-testid="host-internal-path">{internalPath}</strong></p>
      <Link data-testid="host-target-link" href={target}>
        跳转到插件内部路径
      </Link>
    </main>
  )
}
