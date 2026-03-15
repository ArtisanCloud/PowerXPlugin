'use client'

import { useState } from 'react'
import { invokeCapability } from '@/lib/api/capabilities'
import { ApiError } from '@/lib/api/normalizeApiError'

export default function TestCapabilityPage() {
  const [status, setStatus] = useState('idle')
  const [trace, setTrace] = useState('pending')
  const [payload, setPayload] = useState('{}')
  const [errorText, setErrorText] = useState('none')

  const handleInvoke = async (kind: 'success' | 'fail' | 'mock') => {
    setStatus('running')
    setErrorText('none')
    setTrace(`trace-${Date.now()}`)

    try {
      const response = await invokeCapability({ kind })
      setPayload(JSON.stringify(response, null, 2))
      setStatus('succeeded')
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'invoke failed'
      setPayload('{}')
      setStatus('failed')
      setErrorText(message)
    }
  }

  return (
    <main style={{ padding: 24 }} data-testid="capability-playground">
      <h1>测试能力调用</h1>
      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <button data-testid="trigger-success" onClick={() => void handleInvoke('success')}>触发成功</button>
        <button data-testid="trigger-fail" onClick={() => void handleInvoke('fail')}>触发失败</button>
        <button data-testid="trigger-mock" onClick={() => void handleInvoke('mock')}>触发模拟</button>
        <button data-testid="trigger-local-debug" onClick={() => setPayload('{"local":true}')}>
          本地调试
        </button>
      </div>
      <p>trace: <strong data-testid="trace-output">{trace}</strong></p>
      <p>status: <strong data-testid="status-indicator">{status}</strong></p>
      <p>error: <strong data-testid="error-indicator">{errorText === 'none' ? 'none' : 'error'}</strong></p>
      <pre data-testid="payload-viewer">{payload}</pre>
      <p>local-trace: <strong data-testid="local-trace">{trace}</strong></p>
      <p>local-status: <strong data-testid="local-status">{status}</strong></p>
      <p>local-error: <strong data-testid="local-error">{errorText}</strong></p>
      <pre data-testid="local-request-preview">{"{\"kind\":\"...\"}"}</pre>
      <pre data-testid="local-response-preview">{payload}</pre>
    </main>
  )
}
