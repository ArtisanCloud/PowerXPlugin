'use client'

import { useSyncExternalStore } from 'react'
import type { Template } from '../api/template'

type AuthState = {
  isAuthenticated: boolean
  accessToken?: string
  expiresAt?: number
}

type TemplateState = {
  items: Template[]
  total: number
  loading: boolean
  auth: AuthState
}

const initialState: TemplateState = {
  items: [],
  total: 0,
  loading: false,
  auth: {
    isAuthenticated: false,
  },
}

let state: TemplateState = { ...initialState }
const listeners = new Set<() => void>()

function emit() {
  listeners.forEach((listener) => listener())
}

function setState(next: Partial<TemplateState>) {
  state = {
    ...state,
    ...next,
  }
  emit()
}

export function setTemplateLoading(loading: boolean): void {
  setState({ loading })
}

export function setTemplates(items: Template[], total?: number): void {
  setState({
    items,
    total: typeof total === 'number' ? total : items.length,
  })
}

export function upsertTemplate(item: Template): void {
  const index = state.items.findIndex((existing) => existing.id === item.id)
  if (index === -1) {
    const items = [item, ...state.items]
    setState({ items, total: items.length })
    return
  }

  const items = [...state.items]
  items[index] = item
  setState({ items })
}

export function removeTemplate(id: number): void {
  const items = state.items.filter((item) => item.id !== id)
  setState({ items, total: items.length })
}

export function setAuthState(payload: AuthState): void {
  setState({ auth: payload })
}

export function resetTemplateStore(): void {
  state = { ...initialState }
  emit()
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function useTemplateStore(): TemplateState {
  return useSyncExternalStore(subscribe, () => state, () => state)
}
