import { apiGet, apiPost } from "./_client";
import type { ApiResponse } from "./_base";

export interface MetadataPagination {
  total: number;
  page: number;
  page_size: number;
}

export interface MetadataPage<T> {
  items: T[];
  pagination: MetadataPagination;
  total: number;
  page: number;
  page_size: number;
}

export interface MetadataDisplay {
  display_name?: string;
  display_description?: string;
}

export interface DictionaryNamespace extends MetadataDisplay {
  uuid: string;
  namespace: string;
  module: string;
  status: string;
  item_count?: number;
}

export interface DictionaryItem extends MetadataDisplay {
  uuid: string;
  namespace_uuid: string;
  code: string;
  status: string;
  sort_order: number;
  reference_count: number;
}

export interface Taxonomy extends MetadataDisplay {
  uuid: string;
  namespace: string;
  module: string;
  max_depth: number;
  status: string;
}

export interface TaxonomyNode extends MetadataDisplay {
  uuid: string;
  taxonomy_uuid: string;
  parent_uuid?: string;
  code: string;
  path: string;
  depth: number;
  status: string;
  reference_count: number;
}

export interface MetadataTag extends MetadataDisplay {
  uuid: string;
  namespace: string;
  resource_type: string;
  code: string;
  color?: string;
  status: string;
  usage_count: number;
}

export interface ResourceType extends MetadataDisplay {
  uuid: string;
  resource_type: string;
  module: string;
  binding_enabled: boolean;
  validator_status: string;
  status: string;
}

export interface MetadataQuery {
  module?: string;
  namespace?: string;
  resource_type?: string;
  status?: string;
  q?: string;
  locale?: string;
  page?: number;
  page_size?: number;
}

export type I18nMap = Record<string, string>;

export interface CreateDictionaryNamespacePayload {
  namespace: string;
  module: string;
  name_i18n: I18nMap;
  description_i18n?: I18nMap;
}

export interface CreateDictionaryItemPayload {
  code: string;
  label_i18n: I18nMap;
  description_i18n?: I18nMap;
  sort_order?: number;
}

export interface CreateTaxonomyPayload extends CreateDictionaryNamespacePayload {
  max_depth: number;
}

export interface CreateTaxonomyNodePayload {
  parent_uuid?: string | null;
  code: string;
  label_i18n: I18nMap;
  description_i18n?: I18nMap;
  sort_order?: number;
}

export interface CreateTagPayload {
  namespace: string;
  resource_type: string;
  code: string;
  color?: string;
  label_i18n: I18nMap;
  description_i18n?: I18nMap;
}

export interface CreateResourceTypePayload {
  resource_type: string;
  module: string;
  name_i18n: I18nMap;
  description_i18n?: I18nMap;
  validator_key?: string;
  binding_enabled: boolean;
}

const cleanQuery = (query: MetadataQuery = {}) =>
  Object.fromEntries(
    Object.entries(query).filter(([, value]) => value !== undefined && value !== "" && value !== "__all__")
  );

export function useMetadataGovernanceApi() {
  const mode = (init?: any) =>
    apiGet<ApiResponse<Record<string, any>>>("admin/metadata/mode", undefined, init);

  const listDictionaryNamespaces = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<DictionaryNamespace>>>("admin/metadata/dictionaries", cleanQuery(query), init);

  const createDictionaryNamespace = (payload: CreateDictionaryNamespacePayload, init?: any) =>
    apiPost<ApiResponse<{ payload: DictionaryNamespace }>>("admin/metadata/dictionaries", payload, init);

  const listDictionaryItems = (namespaceUuid: string, query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<DictionaryItem>>>(
      `admin/metadata/dictionaries/${encodeURIComponent(namespaceUuid)}/items`,
      cleanQuery(query),
      init
    );

  const createDictionaryItem = (namespaceUuid: string, payload: CreateDictionaryItemPayload, init?: any) =>
    apiPost<ApiResponse<{ payload: DictionaryItem }>>(
      `admin/metadata/dictionaries/${encodeURIComponent(namespaceUuid)}/items`,
      payload,
      init
    );

  const listTaxonomies = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<Taxonomy>>>("admin/metadata/taxonomies", cleanQuery(query), init);

  const createTaxonomy = (payload: CreateTaxonomyPayload, init?: any) =>
    apiPost<ApiResponse<{ payload: Taxonomy }>>("admin/metadata/taxonomies", payload, init);

  const listTaxonomyNodes = (taxonomyUuid: string, query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<TaxonomyNode>>>(
      `admin/metadata/taxonomies/${encodeURIComponent(taxonomyUuid)}/nodes`,
      cleanQuery(query),
      init
    );

  const createTaxonomyNode = (taxonomyUuid: string, payload: CreateTaxonomyNodePayload, init?: any) =>
    apiPost<ApiResponse<{ payload: TaxonomyNode }>>(
      `admin/metadata/taxonomies/${encodeURIComponent(taxonomyUuid)}/nodes`,
      payload,
      init
    );

  const listTags = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<MetadataTag>>>("admin/metadata/tags", cleanQuery(query), init);

  const createTag = (payload: CreateTagPayload, init?: any) =>
    apiPost<ApiResponse<{ payload: MetadataTag }>>("admin/metadata/tags", payload, init);

  const listResourceTypes = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<ResourceType>>>("admin/metadata/resource-types", cleanQuery(query), init);

  const createResourceType = (payload: CreateResourceTypePayload, init?: any) =>
    apiPost<ApiResponse<{ payload: ResourceType }>>("admin/metadata/resource-types", payload, init);

  return {
    mode,
    listDictionaryNamespaces,
    createDictionaryNamespace,
    listDictionaryItems,
    createDictionaryItem,
    listTaxonomies,
    createTaxonomy,
    listTaxonomyNodes,
    createTaxonomyNode,
    listTags,
    createTag,
    listResourceTypes,
    createResourceType,
  };
}
