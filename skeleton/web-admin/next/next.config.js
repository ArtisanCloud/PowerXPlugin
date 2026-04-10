/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  distDir: '.output',
  async rewrites() {
    const apiProxy = process.env.NEXT_DEV_API_PROXY
    if (!apiProxy) {
      return []
    }

    return [
      {
        source: '/api/:path*',
        destination: `${apiProxy}/api/:path*`,
      },
      {
        source: '/_p/:pluginId/api/:path*',
        destination: `${apiProxy}/_p/:pluginId/api/:path*`,
      },
    ]
  },
}

module.exports = nextConfig
