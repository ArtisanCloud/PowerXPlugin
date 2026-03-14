'use client'

import { useEffect, useState } from 'react'

export type TemplateFormValue = {
  name: string
  description: string
  content: string
}

type TemplateFormModalProps = {
  open: boolean
  title: string
  submitLabel: string
  loading?: boolean
  initialValue?: Partial<TemplateFormValue>
  onClose: () => void
  onSubmit: (value: TemplateFormValue) => void
}

const EMPTY_FORM: TemplateFormValue = {
  name: '',
  description: '',
  content: '',
}

export default function TemplateFormModal({
  open,
  title,
  submitLabel,
  loading = false,
  initialValue,
  onClose,
  onSubmit,
}: TemplateFormModalProps) {
  const [form, setForm] = useState<TemplateFormValue>(EMPTY_FORM)

  useEffect(() => {
    if (!open) return
    setForm({
      name: initialValue?.name || '',
      description: initialValue?.description || '',
      content: initialValue?.content || '',
    })
  }, [open, initialValue?.name, initialValue?.description, initialValue?.content])

  if (!open) {
    return null
  }

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    if (!form.name.trim() || !form.description.trim() || !form.content.trim()) {
      return
    }
    onSubmit({
      name: form.name.trim(),
      description: form.description.trim(),
      content: form.content,
    })
  }

  return (
    <div
      aria-modal="true"
      role="dialog"
      data-testid="template-form-modal"
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(15,23,42,0.45)',
        display: 'grid',
        placeItems: 'center',
        padding: 16,
        zIndex: 1000,
      }}
    >
      <div
        style={{
          width: 'min(880px, 100%)',
          borderRadius: 12,
          background: '#fff',
          boxShadow: '0 20px 55px rgba(2, 6, 23, 0.2)',
          border: '1px solid #e2e8f0',
        }}
      >
        <div style={{ padding: '16px 20px', borderBottom: '1px solid #e2e8f0' }}>
          <h2 style={{ margin: 0, fontSize: 18 }}>{title}</h2>
        </div>
        <form onSubmit={handleSubmit} style={{ padding: 20 }}>
          <div style={{ display: 'grid', gap: 14 }}>
            <label>
              名称
              <input
                data-testid="template-form-name"
                value={form.name}
                disabled={loading}
                onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))}
                style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }}
              />
            </label>
            <label>
              描述
              <textarea
                data-testid="template-form-description"
                value={form.description}
                disabled={loading}
                rows={3}
                onChange={(event) =>
                  setForm((prev) => ({ ...prev, description: event.target.value }))
                }
                style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }}
              />
            </label>
            <label>
              内容
              <textarea
                data-testid="template-form-content"
                value={form.content}
                disabled={loading}
                rows={8}
                onChange={(event) => setForm((prev) => ({ ...prev, content: event.target.value }))}
                style={{ display: 'block', width: '100%', marginTop: 6, padding: 10 }}
              />
            </label>
          </div>

          <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
            <button type="button" onClick={onClose} disabled={loading} data-testid="template-form-cancel">
              取消
            </button>
            <button type="submit" disabled={loading} data-testid="template-form-submit">
              {loading ? '提交中...' : submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
