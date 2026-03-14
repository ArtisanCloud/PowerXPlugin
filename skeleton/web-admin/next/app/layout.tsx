import type { ReactNode } from 'react'

export const metadata = {
  title: 'PowerX Plugin',
  description: 'PowerX Plugin Admin',
}

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  )
}
