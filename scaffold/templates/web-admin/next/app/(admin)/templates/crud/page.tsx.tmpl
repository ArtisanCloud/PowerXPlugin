'use client'

import { useEffect, useMemo, useState } from 'react'
import TemplateFormModal, { type TemplateFormValue } from '@/components/templates/TemplateFormModal'
import { tAdmin } from '@/lib/i18n/admin'
import {
  createTemplate,
  deleteTemplate,
  listTemplates,
  type Template,
  updateTemplate,
} from '@/lib/api/template'
import { ApiError } from '@/lib/api/normalizeApiError'
import {
  removeTemplate,
  setTemplateLoading,
  setTemplates,
  upsertTemplate,
  useTemplateStore,
} from '@/lib/stores/templates'
import { useLocalePreference } from '@/lib/ui/preferences'

function toPayload(value: TemplateFormValue) {
  return {
    name: value.name.trim(),
    description: value.description.trim(),
    content: value.content,
  }
}

export default function TemplatesCrudPage() {
  const store = useTemplateStore()
  const locale = useLocalePreference()
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Template | null>(null)

  useEffect(() => {
    let active = true
    const load = async () => {
      setTemplateLoading(true)
      setError('')
      try {
        const data = await listTemplates(1, 50)
        if (!active) return
        setTemplates(data.list, data.total)
      } catch (err) {
        if (!active) return
        if (err instanceof ApiError) {
          setError(err.message)
        } else {
          setError(tAdmin(locale, 'templates.crud.error.load'))
        }
      } finally {
        if (active) setTemplateLoading(false)
      }
    }

    void load()
    return () => {
      active = false
    }
  }, [locale])

  const modalTitle = useMemo(
    () => (editing ? tAdmin(locale, 'templates.crud.modal.editTitle') : tAdmin(locale, 'templates.crud.modal.createTitle')),
    [editing, locale],
  )
  const submitLabel = useMemo(
    () => (editing ? tAdmin(locale, 'templates.crud.modal.editSubmit') : tAdmin(locale, 'templates.crud.modal.createSubmit')),
    [editing, locale],
  )

  const handleSubmit = async (value: TemplateFormValue) => {
    setSaving(true)
    setError('')
    try {
      const payload = toPayload(value)
      if (editing) {
        const updated = await updateTemplate(editing.id, payload)
        upsertTemplate(updated)
      } else {
        const created = await createTemplate(payload)
        upsertTemplate(created)
      }
      setModalOpen(false)
      setEditing(null)
    } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message)
        } else {
          setError(tAdmin(locale, 'templates.crud.error.save'))
        }
      } finally {
        setSaving(false)
      }
  }

  const handleDelete = async (item: Template) => {
    const ok = window.confirm(tAdmin(locale, 'templates.crud.confirm.delete').replace('{name}', item.name))
    if (!ok) return

    setError('')
    try {
      await deleteTemplate(item.id)
      removeTemplate(item.id)
    } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message)
        } else {
          setError(tAdmin(locale, 'templates.crud.error.delete'))
        }
      }
  }

  return (
    <main className="px-admin-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <div className="px-admin-toolbar" style={{ justifyContent: 'space-between' }}>
            <div>
              <h1 data-testid="templates-crud-title" className="px-admin-title">{tAdmin(locale, 'templates.crud.title')}</h1>
              <p className="px-admin-subtitle">{tAdmin(locale, 'templates.crud.subtitle')}</p>
            </div>
            <button className="px-btn" data-testid="templates-create-btn" onClick={() => { setEditing(null); setModalOpen(true) }}>
              {tAdmin(locale, 'templates.crud.create')}
            </button>
          </div>

          {error ? (
            <p role="alert" data-testid="templates-crud-error" className="px-alert px-alert-danger" style={{ marginTop: 12 }}>
              {error}
            </p>
          ) : null}

          <div style={{ marginTop: 12 }}>
            <span className="px-badge" data-testid="templates-total">{tAdmin(locale, 'templates.crud.total')}：{store.total}</span>
          </div>

          <div className="px-table-wrap" style={{ marginTop: 14 }}>
            <table data-testid="templates-table" className="px-table">
              <thead>
                <tr>
                  <th>{tAdmin(locale, 'templates.crud.col.name')}</th>
                  <th>{tAdmin(locale, 'templates.crud.col.description')}</th>
                  <th>{tAdmin(locale, 'templates.crud.col.content')}</th>
                  <th>{tAdmin(locale, 'templates.crud.col.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {store.loading ? (
                  <tr>
                    <td colSpan={4} data-testid="templates-loading">{tAdmin(locale, 'templates.crud.loading')}</td>
                  </tr>
                ) : null}

                {!store.loading && store.items.length === 0 ? (
                  <tr>
                    <td colSpan={4} data-testid="templates-empty">{tAdmin(locale, 'templates.crud.empty')}</td>
                  </tr>
                ) : null}

                {store.items.map((item) => (
                  <tr key={item.id} data-testid={`template-row-${item.id}`}>
                    <td>{item.name}</td>
                    <td>{item.description}</td>
                    <td>{item.content}</td>
                    <td>
                      <div className="px-row-actions">
                        <button className="px-btn-ghost" data-testid={`template-edit-${item.id}`} onClick={() => { setEditing(item); setModalOpen(true) }}>
                          {tAdmin(locale, 'templates.crud.edit')}
                        </button>
                        <button className="px-btn-danger" data-testid={`template-delete-${item.id}`} onClick={() => void handleDelete(item)}>
                          {tAdmin(locale, 'templates.crud.delete')}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <TemplateFormModal
            open={modalOpen}
            title={modalTitle}
            submitLabel={submitLabel}
            loading={saving}
            initialValue={editing || undefined}
            locale={locale}
            onClose={() => {
              setModalOpen(false)
              setEditing(null)
            }}
            onSubmit={(value) => void handleSubmit(value)}
          />
        </article>
      </section>
    </main>
  )
}
