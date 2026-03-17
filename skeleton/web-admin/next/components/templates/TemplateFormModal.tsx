'use client'

import { useEffect, useState } from 'react'
import { tAdmin } from '@/lib/i18n/admin'
import type { AdminLocale } from '@/lib/ui/preferences'

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
  locale: AdminLocale
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
  locale,
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

  if (!open) return null

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    if (!form.name.trim() || !form.description.trim() || !form.content.trim()) return
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
        background: 'rgba(2, 6, 23, 0.7)',
        display: 'grid',
        placeItems: 'center',
        padding: 16,
        zIndex: 1000,
      }}
    >
      <div style={{ width: 'min(780px, 100%)', borderRadius: 12, background: '#0b1738', border: '1px solid #1d2f5c', color: '#e2e8f0', boxShadow: '0 20px 45px rgba(2,6,23,.35)' }}>
        <div style={{ padding: '16px 20px', borderBottom: '1px solid #243b6b' }}>
          <h2 style={{ margin: 0, fontSize: 18 }}>{title}</h2>
        </div>

        <form onSubmit={handleSubmit} style={{ padding: 20, display: 'grid', gap: 12 }}>
          <label className="px-form-row">
            <span className="px-form-label">{tAdmin(locale, 'templates.crud.modal.name')}</span>
            <input data-testid="template-form-name" className="px-input" value={form.name} disabled={loading} onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))} />
          </label>

          <label className="px-form-row">
            <span className="px-form-label">{tAdmin(locale, 'templates.crud.modal.description')}</span>
            <textarea data-testid="template-form-description" className="px-input" style={{ height: 96, paddingTop: 8 }} value={form.description} disabled={loading} onChange={(event) => setForm((prev) => ({ ...prev, description: event.target.value }))} />
          </label>

          <label className="px-form-row">
            <span className="px-form-label">{tAdmin(locale, 'templates.crud.modal.content')}</span>
            <textarea data-testid="template-form-content" className="px-input" style={{ height: 180, paddingTop: 8 }} value={form.content} disabled={loading} onChange={(event) => setForm((prev) => ({ ...prev, content: event.target.value }))} />
          </label>

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
            <button type="button" className="px-btn-ghost" onClick={onClose} disabled={loading} data-testid="template-form-cancel">{tAdmin(locale, 'templates.crud.modal.cancel')}</button>
            <button type="submit" className="px-btn" disabled={loading} data-testid="template-form-submit">{loading ? tAdmin(locale, 'templates.crud.modal.submitting') : submitLabel}</button>
          </div>
        </form>
      </div>
    </div>
  )
}
