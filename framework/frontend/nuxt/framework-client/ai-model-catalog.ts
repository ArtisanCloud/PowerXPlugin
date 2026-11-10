export type LocalCatalogModality = 'llm' | 'vlm' | 'embedding' | 'image' | 'video' | 'audio_tts' | 'audio_asr' | 'model3d' | 'rerank';
export interface LocalOnlineCatalogInput {
  environment: 'development' | 'production'; modality: LocalCatalogModality;
  provider: string; app: string; endpoint: string; credential_ref: string;
}
export interface OnlineCatalogModel {
  id: string; label: string; modalities: string[];
  match: 'unmatched' | 'local_catalog' | 'provider_declared' | 'manual'; manual_modalities: string[];
}
export interface LocalOnlineCatalog {
  contract_version: 1; uuid: string; environment: 'development' | 'production'; provider: string; app: string;
  revision: string; protocol: string; source: string; updated_at: string; models: OnlineCatalogModel[];
}
export class AIModelCatalogContractError extends Error {
  constructor(public readonly reasonCode = 'AI_MODEL_CATALOG_RESPONSE_INVALID') { super(reasonCode); }
}
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
export function validateLocalOnlineCatalog(value: unknown, input: LocalOnlineCatalogInput): LocalOnlineCatalog {
  const result = value as LocalOnlineCatalog;
  if (result && Object.keys(result).some(key => !['contract_version', 'uuid', 'environment', 'provider', 'app', 'revision', 'protocol', 'source', 'updated_at', 'models'].includes(key))) throw new AIModelCatalogContractError();
  if (!result || result.contract_version !== 1 || !uuid.test(result.uuid) || !uuid.test(result.revision) || result.environment !== input.environment || result.provider !== input.provider || result.app !== input.app || typeof result.protocol !== 'string' || !result.protocol || typeof result.source !== 'string' || !Number.isFinite(Date.parse(result.updated_at)) || !Array.isArray(result.models) || result.models.length > 10000) throw new AIModelCatalogContractError();
  let source: URL; try { source = new URL(result.source); } catch { throw new AIModelCatalogContractError(); }
  if (!['http:', 'https:'].includes(source.protocol) || source.username || source.password || source.search || source.hash) throw new AIModelCatalogContractError();
  const ids = new Set<string>();
  for (const model of result.models) {
    if (typeof model.id !== 'string' || !model.id || model.id !== model.id.trim() || model.id.length > 512 || /[\r\n\0]/.test(model.id) || ids.has(model.id) || typeof model.label !== 'string' || !model.label || !Array.isArray(model.modalities) || model.modalities.some(m => typeof m !== 'string') || !Array.isArray(model.manual_modalities) || model.manual_modalities.some(m => !['llm', 'embedding'].includes(m)) || !['unmatched', 'local_catalog', 'provider_declared', 'manual'].includes(model.match)) throw new AIModelCatalogContractError();
    ids.add(model.id);
  }
  return result;
}
export function createLocalModelCatalogClient(options: { request<T>(path: string, options: { method: 'POST'; body: LocalOnlineCatalogInput & { action: 'refresh' | 'assign'; revision?: string; models?: string[] } }): Promise<{ data: T }> }) {
  return {
    async sync(input: LocalOnlineCatalogInput, assignment?: { revision: string; models: string[] }) {
      if (!input || !['development', 'production'].includes(input.environment) || !['llm', 'vlm', 'embedding', 'image', 'video', 'audio_tts', 'audio_asr', 'model3d', 'rerank'].includes(input.modality) || typeof input.provider !== 'string' || !input.provider || input.provider !== input.provider.trim() || typeof input.app !== 'string' || typeof input.endpoint !== 'string' || typeof input.credential_ref !== 'string' || Object.keys(input).some(key => !['environment', 'modality', 'provider', 'app', 'endpoint', 'credential_ref'].includes(key))) throw new AIModelCatalogContractError('AI_REQUEST_INVALID');
      if (assignment && (!['llm', 'embedding'].includes(input.modality) || !uuid.test(assignment.revision) || !Array.isArray(assignment.models) || !assignment.models.length || assignment.models.length > 1000 || new Set(assignment.models).size !== assignment.models.length)) throw new AIModelCatalogContractError('AI_REQUEST_INVALID');
      if (assignment && (Object.keys(assignment).some(key => !['revision', 'models'].includes(key)) || assignment.models.some(id => typeof id !== 'string' || !id || id.length > 512 || id !== id.trim() || /[\r\n\0]/.test(id)))) throw new AIModelCatalogContractError('AI_REQUEST_INVALID');
      const result = await options.request<LocalOnlineCatalog>('admin/ai-settings/catalog/local-online', { method: 'POST', body: { ...input, action: assignment ? 'assign' : 'refresh', ...(assignment || {}) } });
      return validateLocalOnlineCatalog(result?.data, input);
    },
  };
}
