'use client'

import { useSyncExternalStore } from 'react'

export type RuntimeMode = 'standalone' | 'host'

export type PermissionState = {
  mode: RuntimeMode
  delegatedIAM: boolean
  permissions: string[]
}

const state: PermissionState = {
  mode: 'standalone',
  delegatedIAM: false,
  permissions: [],
}

const listeners = new Set<() => void>()

function emit() {
  listeners.forEach((listener) => listener())
}

export function setPermissionMode(mode: RuntimeMode): void {
  state.mode = mode
  emit()
}

export function setDelegatedIAM(enabled: boolean): void {
  state.delegatedIAM = enabled
  emit()
}

export function setPermissions(permissions: string[]): void {
  state.permissions = [...permissions]
  emit()
}

export function canAccess(resource: string): boolean {
  if (!state.delegatedIAM) return true
  return state.permissions.includes(resource)
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function usePermissionStore(): PermissionState {
  return useSyncExternalStore(subscribe, () => state, () => state)
}
