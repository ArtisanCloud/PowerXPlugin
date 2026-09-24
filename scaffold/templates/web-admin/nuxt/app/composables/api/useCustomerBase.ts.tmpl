import { apiGet, apiPatch, apiPost, useApiClient } from "./_client";
import type { ApiResponse } from "./_base";
import type { ProviderModeDiagnostics } from "./useProviderMode";

export interface CustomerModeDiagnostics extends ProviderModeDiagnostics {
  gateway_auth_scheme?: string;
  account_selector_capability?: string;
  account_manage_capability?: string;
  contact_read_capability?: string;
  contact_manage_capability?: string;
}

export interface CustomerOverview {
  tenant_uuid?: string;
  accounts_total: number;
  accounts_active: number;
  memberships_total: number;
  memberships_active: number;
  mini_app_entries_active: number;
  login_events_24h: number;
}

export interface CustomerPage<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface CustomerAccount {
  id: number;
  customer_uuid: string;
  tenant_uuid?: string;
  primary_email?: string;
  primary_phone?: string;
  email?: string;
  phone?: string;
  display_name?: string;
  nickname?: string;
  given_name?: string;
  family_name?: string;
  avatar_url?: string;
  locale?: string;
  timezone?: string;
  status: string;
  email_verified: boolean;
  phone_verified: boolean;
  metadata?: Record<string, any>;
  created_at?: string;
  updated_at?: string;
}

export interface CustomerIdentity {
  id: number;
  customer_uuid: string;
  provider: string;
  provider_subject?: string;
  email?: string;
  phone?: string;
  status: string;
  verified_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CustomerContact {
  contact_uuid: string;
  customer_uuid: string;
  display_name: string;
  given_name?: string;
  family_name?: string;
  status: "active" | "inactive" | "temporary";
  roles: ("primary" | "legal_representative")[];
  tags: string[];
  created_at?: string;
  updated_at?: string;
}

export interface CustomerContactIdentity {
  identity_uuid: string;
  contact_uuid: string;
  channel_dictionary_item_uuid: string;
  external_subject: string;
  status: "active" | "inactive";
}

export interface CreateCustomerContactInput {
  display_name: string;
  given_name?: string;
  family_name?: string;
  status: "active" | "inactive" | "temporary";
  roles?: ("primary" | "legal_representative")[];
  tags?: string[];
  creation_intent: "explicit_create" | "explicit_temporary";
}

export interface UpdateCustomerContactInput {
  display_name?: string;
  given_name?: string;
  family_name?: string;
  status?: "active" | "inactive" | "temporary";
  roles?: ("primary" | "legal_representative")[];
  tags?: string[];
}

export interface CustomerMembership {
  id: number;
  membership_uuid: string;
  tenant_uuid: string;
  customer_uuid: string;
  status: string;
  roles?: unknown;
  scopes?: unknown;
  source: string;
  expires_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CustomerLoginEvent {
  id: number;
  tenant_uuid?: string;
  customer_uuid?: string;
  identity_provider?: string;
  event_type: string;
  ok: boolean;
  error_code?: string;
  ip?: string;
  user_agent?: string;
  trace_id?: string;
  created_at?: string;
}

export interface MiniAppEntry {
  id: number;
  entry_uuid: string;
  tenant_uuid: string;
  entry_code: string;
  entry_type: string;
  app_key?: string;
  appid?: string;
  channel?: string;
  campaign?: string;
  brand_name?: string;
  org_name?: string;
  status: string;
  expires_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CustomerBaseQuery {
  page?: number;
  page_size?: number;
  q?: string;
  status?: string;
  provider?: string;
  framework_debug_route?: "local" | "delegated";
  customer_uuid?: string;
  tenant_uuid?: string;
}

export interface CreateCustomerAccountInput {
  tenant_uuid: string;
  email?: string;
  phone?: string;
  password: string;
  display_name?: string;
  nickname?: string;
  given_name?: string;
  family_name?: string;
  avatar_url?: string;
  locale?: string;
  timezone?: string;
  metadata?: Record<string, any>;
}

export interface CreateBasicCustomerInput {
  display_name: string;
  nickname?: string;
  given_name?: string;
  family_name?: string;
  primary_email?: string;
  primary_phone?: string;
  avatar_url?: string;
  status?: "active" | "pending" | "suspended" | "disabled";
  locale?: string;
  timezone?: string;
}

export interface CreateCustomerAccountResult {
  customer_uuid: string;
  tenant_uuid: string;
  status: string;
}

export interface UpdateCustomerAccountInput {
  primary_email?: string;
  primary_phone?: string;
  email?: string;
  phone?: string;
  display_name?: string;
  nickname?: string;
  given_name?: string;
  family_name?: string;
  avatar_url?: string;
  locale?: string;
  timezone?: string;
  status?: string;
  email_verified?: boolean;
  phone_verified?: boolean;
  metadata?: Record<string, any>;
}

const cleanQuery = (query: CustomerBaseQuery = {}) =>
  Object.fromEntries(
    Object.entries(query).filter(([, value]) => value !== undefined && value !== "")
  );

export function useCustomerBaseApi() {
  const { baseURL } = useApiClient();

  const overview = (query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerOverview>>("admin/customers/overview", cleanQuery(query), init);

  const mode = (init?: any) =>
    apiGet<ApiResponse<CustomerModeDiagnostics>>("admin/customers/mode", undefined, init);

  const listAccounts = (query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerPage<CustomerAccount>>>("admin/customers/accounts", cleanQuery(query), init);

  const createAccount = (input: CreateCustomerAccountInput, init?: any) =>
    apiPost<ApiResponse<CreateCustomerAccountResult>>("admin/customers/accounts", input, init);

  const createBasicAccount = (input: CreateBasicCustomerInput, route: "local" | "delegated", init?: any) =>
    apiPost<ApiResponse<CustomerAccount>>(`admin/customers/debug/basic-accounts?framework_debug_route=${route}`, input, init);

  const getAccount = (customerUUID: string, query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerAccount>>(`admin/customers/accounts/${encodeURIComponent(customerUUID)}`, cleanQuery(query), init);

  const updateAccount = (customerUUID: string, input: UpdateCustomerAccountInput, init?: any) =>
    apiPatch<ApiResponse<CustomerAccount>>(`admin/customers/accounts/${encodeURIComponent(customerUUID)}`, input, init);

  const listIdentities = (query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerPage<CustomerIdentity>>>("admin/customers/identities", cleanQuery(query), init);

  const listMemberships = (query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerPage<CustomerMembership>>>("admin/customers/memberships", cleanQuery(query), init);

  const listLoginEvents = (query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerPage<CustomerLoginEvent>>>("admin/customers/login-events", cleanQuery(query), init);

  const listMiniAppEntries = (query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerPage<MiniAppEntry>>>("admin/customers/mini-app-entries", cleanQuery(query), init);

  const listContacts = (customerUUID: string, query?: CustomerBaseQuery, init?: any) =>
    apiGet<ApiResponse<CustomerPage<CustomerContact>>>(`admin/customers/${encodeURIComponent(customerUUID)}/contacts`, cleanQuery(query), init);

  const createContact = (customerUUID: string, input: CreateCustomerContactInput, init?: any) =>
    apiPost<ApiResponse<CustomerContact>>(`admin/customers/${encodeURIComponent(customerUUID)}/contacts`, input, init);

  const updateContact = (customerUUID: string, contactUUID: string, input: UpdateCustomerContactInput, init?: any) =>
    apiPatch<ApiResponse<CustomerContact>>(`admin/customers/${encodeURIComponent(customerUUID)}/contacts/${encodeURIComponent(contactUUID)}`, input, init);

  const bindContactIdentity = (customerUUID: string, contactUUID: string, input: { channel_dictionary_item_uuid: string; external_subject: string }, init?: any) =>
    apiPost<ApiResponse<CustomerContactIdentity>>(`admin/customers/${encodeURIComponent(customerUUID)}/contacts/${encodeURIComponent(contactUUID)}/identities`, input, init);

  return {
    baseURL,
    mode,
    overview,
    listAccounts,
    createAccount,
    createBasicAccount,
    getAccount,
    updateAccount,
    listIdentities,
    listMemberships,
    listLoginEvents,
    listMiniAppEntries,
    listContacts,
    createContact,
    updateContact,
    bindContactIdentity,
  };
}
