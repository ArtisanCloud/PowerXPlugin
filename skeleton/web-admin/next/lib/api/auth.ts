import { apiRequest } from './client'

export type LoginRequest = {
  tenant?: string
  identifier: string
  password: string
  remember?: boolean
}

export type LoginResponse = {
  token_type?: string
  access_token: string
  refresh_token: string
  expires_in?: number
  expires_at: number
  scope?: string
}

export type RegisterRequest = {
  tenant_uuid?: string
  username: string
  email: string
  password: string
  display_name?: string
}

export type RegisterResponse = {
  user?: Record<string, unknown>
  member?: Record<string, unknown>
}

export type ForgotPasswordRequest = {
  email: string
}

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  return apiRequest<LoginResponse>('/admin/user/auth/login', {
    method: 'POST',
    body: payload,
  })
}

export async function register(payload: RegisterRequest): Promise<RegisterResponse> {
  return apiRequest<RegisterResponse>('/admin/user/auth/register', {
    method: 'POST',
    body: payload,
  })
}

export async function requestPasswordReset(payload: ForgotPasswordRequest): Promise<Record<string, unknown>> {
  return apiRequest<Record<string, unknown>>('/admin/user/auth/reset-password', {
    method: 'POST',
    body: payload,
  })
}
