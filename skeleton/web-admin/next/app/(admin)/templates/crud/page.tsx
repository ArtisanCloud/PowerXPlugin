'use client'

import { useEffect, useMemo, useState } from 'react'
import TemplateFormModal, { type TemplateFormValue } from '@/components/templates/TemplateFormModal'
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

function toPayload(value: TemplateFormValue) {
  return {
    name: value.name.trim(),
    description: value.description.trim(),
    content: value.content,
  }
}

export default function TemplatesCrudPage() {
  const store = useTemplateStore()
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
          setError('加载模板失败，请稍后重试。')
        }
      } finally {
        if (active) {
          setTemplateLoading(false)
        }
      }
    }

    void load()
    return () => {
      active = false
    }
  }, [])

  const modalTitle = useMemo(() => (editing ? '编辑模板' : '创建模板'), [editing])
  const submitLabel = useMemo(() => (editing ? '保存修改' : '创建模板'), [editing])

  const startCreate = () => {
    setEditing(null)
    setModalOpen(true)
  }

  const startEdit = (item: Template) => {
    setEditing(item)
    setModalOpen(true)
  }

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
        setError('保存失败，请稍后重试。')
      }
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (item: Template) => {
    const ok = window.confirm(`确认删除模板「${item.name}」？`)
    if (!ok) return

    setError('')
    try {
      await deleteTemplate(item.id)
      removeTemplate(item.id)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('删除失败，请稍后重试。')
      }
    }
  }

  return (
    <main style={{ padding: 24, display: 'grid', gap: 14 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
        <div>
          <h1 data-testid="templates-crud-title" style={{ margin: 0 }}>模板 CRUD</h1>
          <p style={{ margin: '8px 0 0', color: '#475569' }}>对齐 Nuxt 基线：创建、编辑、删除模板。</p>
        </div>
        <button data-testid="templates-create-btn" onClick={startCreate}>创建模板</button>
      </div>

      {error ? (
        <p role="alert" data-testid="templates-crud-error" style={{ color: '#b91c1c' }}>{error}</p>
      ) : null}

      <div data-testid="templates-total">总数：{store.total}</div>

      <table data-testid="templates-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={{ textAlign: 'left', borderBottom: '1px solid #cbd5e1', padding: '8px 6px' }}>名称</th>
            <th style={{ textAlign: 'left', borderBottom: '1px solid #cbd5e1', padding: '8px 6px' }}>描述</th>
            <th style={{ textAlign: 'left', borderBottom: '1px solid #cbd5e1', padding: '8px 6px' }}>内容</th>
            <th style={{ textAlign: 'left', borderBottom: '1px solid #cbd5e1', padding: '8px 6px' }}>操作</th>
          </tr>
        </thead>
        <tbody>
          {store.loading ? (
            <tr>
              <td colSpan={4} style={{ padding: 12 }} data-testid="templates-loading">加载中...</td>
            </tr>
          ) : null}

          {!store.loading && store.items.length === 0 ? (
            <tr>
              <td colSpan={4} style={{ padding: 12 }} data-testid="templates-empty">暂无模板</td>
            </tr>
          ) : null}

          {store.items.map((item) => (
            <tr key={item.id} data-testid={`template-row-${item.id}`}>
              <td style={{ borderBottom: '1px solid #e2e8f0', padding: '8px 6px' }}>{item.name}</td>
              <td style={{ borderBottom: '1px solid #e2e8f0', padding: '8px 6px' }}>{item.description}</td>
              <td style={{ borderBottom: '1px solid #e2e8f0', padding: '8px 6px' }}>{item.content}</td>
              <td style={{ borderBottom: '1px solid #e2e8f0', padding: '8px 6px', display: 'flex', gap: 8 }}>
                <button data-testid={`template-edit-${item.id}`} onClick={() => startEdit(item)}>编辑</button>
                <button data-testid={`template-delete-${item.id}`} onClick={() => void handleDelete(item)}>删除</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <TemplateFormModal
        open={modalOpen}
        title={modalTitle}
        submitLabel={submitLabel}
        loading={saving}
        initialValue={editing || undefined}
        onClose={() => {
          setModalOpen(false)
          setEditing(null)
        }}
        onSubmit={(value) => void handleSubmit(value)}
      />
    </main>
  )
}
