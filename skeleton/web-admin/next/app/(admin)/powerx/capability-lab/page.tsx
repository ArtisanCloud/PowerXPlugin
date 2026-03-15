'use client'

import Link from 'next/link'

export default function CapabilityLabPage() {
  return (
    <main style={{ padding: 24 }} data-testid="capability-lab-page">
      <h1>Capability Lab</h1>
      <p>用于联调能力调用、trace 与错误语义。</p>
      <Link href="/tests/capability">进入测试能力页面</Link>
    </main>
  )
}
