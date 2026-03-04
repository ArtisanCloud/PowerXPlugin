import { defineStore } from 'pinia'
import type { SlaProfile, SlaProfileUpdatePayload, SlaActualsPayload, SlaPlanType } from '~/types/operations'

interface SlaState {
  profiles: SlaProfile[]
  loading: boolean
  error: string | null
}

interface ApiSlaProfile {
  id: string
  plugin_id: string
  plan_type: SlaPlanType
  uptime_target: number
  uptime_actual: number
  response_target_ms: number
  response_actual_ms: number
  success_target_pct: number
  success_actual_pct: number
  support_frt_target_hours: number
  support_frt_actual_hours: number
  sla_score: number
  incentive_applied_at?: string | null
  penalty_applied_at?: string | null
  notes?: string | null
  computed_at: string
  created_at: string
  updated_at: string
}

const toSlaProfile = (profile: ApiSlaProfile): SlaProfile => ({
  id: profile.id,
  pluginId: profile.plugin_id,
  planType: profile.plan_type,
  uptimeTarget: profile.uptime_target,
  uptimeActual: profile.uptime_actual,
  responseTargetMs: profile.response_target_ms,
  responseActualMs: profile.response_actual_ms,
  successTargetPct: profile.success_target_pct,
  successActualPct: profile.success_actual_pct,
  supportFrtTargetHours: profile.support_frt_target_hours,
  supportFrtActualHours: profile.support_frt_actual_hours,
  slaScore: profile.sla_score,
  incentiveAppliedAt: profile.incentive_applied_at,
  penaltyAppliedAt: profile.penalty_applied_at,
  notes: profile.notes ?? undefined,
  computedAt: profile.computed_at,
  createdAt: profile.created_at,
  updatedAt: profile.updated_at,
})

export const useSlaStore = defineStore('operations.sla', {
  state: (): SlaState => ({
    profiles: [],
    loading: false,
    error: null,
  }),
  actions: {
    apiBase() {
      const config = useRuntimeConfig()
      const base = config.public?.apiBaseUrl || '/api/v1'
      return `${base.replace(/\/$/, '')}/admin/operations/sla`
    },
    async fetchProfiles() {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<ApiSlaProfile[]>(`${this.apiBase()}/profiles`, {
          credentials: 'include',
        })
        this.profiles = (response ?? []).map(toSlaProfile)
      } catch (err: any) {
        this.error = err?.message ?? '加载 SLA 配置失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async upsertProfile(payload: SlaProfileUpdatePayload) {
      this.loading = true
      this.error = null
      try {
        const response = await $fetch<ApiSlaProfile>(`${this.apiBase()}/profiles`, {
          method: 'POST',
          credentials: 'include',
          body: payload,
        })
        await this.fetchProfiles()
        return toSlaProfile(response)
      } catch (err: any) {
        this.error = err?.message ?? '更新 SLA 目标失败'
        throw err
      } finally {
        this.loading = false
      }
    },
    async updateActuals(payload: SlaActualsPayload) {
      this.error = null
      try {
        await $fetch<ApiSlaProfile>(`${this.apiBase()}/profiles/actuals`, {
          method: 'PATCH',
          credentials: 'include',
          body: payload,
        })
        await this.fetchProfiles()
      } catch (err: any) {
        this.error = err?.message ?? '更新 SLA 指标失败'
        throw err
      }
    },
    async recompute() {
      this.error = null
      try {
        await $fetch<unknown>(`${this.apiBase()}/profiles/recompute`, {
          method: 'POST',
          credentials: 'include',
        })
      } catch (err: any) {
        this.error = err?.message ?? '触发重算失败'
        throw err
      }
    },
  },
})
