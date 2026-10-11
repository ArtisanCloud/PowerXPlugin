import { apiGet, apiPost, apiPatch } from "./_client";
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

export interface UpdateTagPayload {
  label_i18n: I18nMap;
  description_i18n: I18nMap;
  color: string;
  status: string;
}

export interface MetadataTag extends MetadataDisplay {
  label_i18n: I18nMap;
  description_i18n?: I18nMap;
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

export function useMetadataGovernanceApi(debugRoute?: () => "local" | "delegated" | undefined) {
  const route = () => debugRoute?.();
  const path = (suffix: string) => `admin/metadata/${route() ? "debug/" : ""}${suffix}`;
  const queryFor = (query: MetadataQuery = {}) => ({ ...cleanQuery(query), ...(route() ? { framework_debug_route: route() } : {}) });
  const createOptions = (init?: any) => ({ ...init, query: { ...init?.query, ...(route() ? { framework_debug_route: route() } : {}) } });
  const mode = (init?: any) =>
    apiGet<ApiResponse<Record<string, any>>>("admin/metadata/mode", undefined, init);

  const listDictionaryNamespaces = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<DictionaryNamespace>>>(path("dictionaries"), queryFor(query), init);

  const createDictionaryNamespace = (payload: CreateDictionaryNamespacePayload, init?: any) =>
    apiPost<ApiResponse<{ payload: DictionaryNamespace }>>(path("dictionaries"), payload, createOptions(init));

  const listDictionaryItems = (namespaceUuid: string, query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<DictionaryItem>>>(
      path(`dictionaries/${encodeURIComponent(namespaceUuid)}/items`),
      queryFor(query),
      init
    );

  const createDictionaryItem = (namespaceUuid: string, payload: CreateDictionaryItemPayload, init?: any) =>
    apiPost<ApiResponse<{ payload: DictionaryItem }>>(
      path(`dictionaries/${encodeURIComponent(namespaceUuid)}/items`),
      payload,
      createOptions(init)
    );

  const listTaxonomies = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<Taxonomy>>>(path("taxonomies"), queryFor(query), init);

  const createTaxonomy = (payload: CreateTaxonomyPayload, init?: any) =>
    apiPost<ApiResponse<{ payload: Taxonomy }>>(path("taxonomies"), payload, createOptions(init));

  const listTaxonomyNodes = (taxonomyUuid: string, query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<TaxonomyNode>>>(
      path(`taxonomies/${encodeURIComponent(taxonomyUuid)}/nodes`),
      queryFor(query),
      init
    );

  const createTaxonomyNode = (taxonomyUuid: string, payload: CreateTaxonomyNodePayload, init?: any) =>
    apiPost<ApiResponse<{ payload: TaxonomyNode }>>(
      path(`taxonomies/${encodeURIComponent(taxonomyUuid)}/nodes`),
      payload,
      createOptions(init)
    );

  const listTags = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<MetadataTag>>>(path("tags"), queryFor(query), init);

  const updateTag = (uuid: string, payload: UpdateTagPayload, init?: any) =>
    apiPatch<ApiResponse<MetadataTag>>(path(`tags/${encodeURIComponent(uuid)}`), payload, createOptions(init));

  const createTag = (payload: CreateTagPayload, init?: any) =>
    apiPost<ApiResponse<{ payload: MetadataTag }>>(path("tags"), payload, createOptions(init));

  const listResourceTypes = (query?: MetadataQuery, init?: any) =>
    apiGet<ApiResponse<MetadataPage<ResourceType>>>(path("resource-types"), queryFor(query), init);

  const createResourceType = (payload: CreateResourceTypePayload, init?: any) =>
    apiPost<ApiResponse<{ payload: ResourceType }>>(path("resource-types"), payload, createOptions(init));

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
    updateTag,
    listResourceTypes,
    createResourceType,
  };
}
