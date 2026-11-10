export type AIConnectionModality = 'llm' | 'vlm' | 'image' | 'video' | 'embedding' | 'audio_tts' | 'audio_asr' | 'model3d' | 'rerank';
export interface AIConnectionVerification {
  provider: string;
  probe: 'models' | 'key_info' | 'ollama_tags';
  endpoint_reachable: true;
  api_key_verified: boolean;
  authentication: 'api_key' | 'not_required';
  model_access_verified: false;
  generation_submitted: false;
}
export interface AIConnectionTestResult {
  contract_version: 1; ok: true; env: string; modality: AIConnectionModality;
  verification: AIConnectionVerification;
}
export type AIConnectionTestInput = { modality: AIConnectionModality; provider: string; model?: string } & (
  { source: 'powerx' } | { source: 'local'; endpoint: string; environment: 'development' | 'production' }
);
export class AIConnectionContractError extends Error {
  constructor(public readonly reasonCode = 'AI_CONNECTION_CONTRACT_INVALID') { super(reasonCode); }
}
export function validateAIConnectionTestResult(value: unknown, input: Pick<AIConnectionTestInput, 'provider' | 'modality'>): AIConnectionTestResult {
  const out = value as AIConnectionTestResult;
  const v = out?.verification;
  if (!out || out.contract_version !== 1 || out.ok !== true || typeof out.env !== 'string' || !out.env.trim() || out.modality !== input.modality || !v || v.provider !== input.provider || v.endpoint_reachable !== true || typeof v.api_key_verified !== 'boolean' || v.model_access_verified !== false || v.generation_submitted !== false) throw new AIConnectionContractError();
  if (v.authentication === 'api_key' ? v.api_key_verified !== true || !['models', 'key_info'].includes(v.probe) : v.authentication !== 'not_required' || v.api_key_verified !== false || v.probe !== 'ollama_tags') throw new AIConnectionContractError();
  if (Object.keys(out).some(key => !['contract_version', 'ok', 'env', 'modality', 'verification'].includes(key)) || Object.keys(v).some(key => !['provider', 'probe', 'endpoint_reachable', 'api_key_verified', 'authentication', 'model_access_verified', 'generation_submitted'].includes(key))) throw new AIConnectionContractError();
  return out;
}
export function createAIConnectionClient(options: {
  request<T>(path: string, options: { method: 'POST'; body: AIConnectionTestInput }): Promise<{ data: T }>;
  path?: string;
}) {
  return {
    async testConnection(input: AIConnectionTestInput) {
      const allowed = ['source', 'modality', 'provider', 'model', ...(input?.source === 'local' ? ['endpoint', 'environment'] : [])];
      if (!input || !['local', 'powerx'].includes(input.source) || typeof input.provider !== 'string' || !input.provider.trim() || input.provider !== input.provider.trim() || (input.model !== undefined && typeof input.model !== 'string') || !['llm', 'vlm', 'image', 'video', 'embedding', 'audio_tts', 'audio_asr', 'model3d', 'rerank'].includes(input.modality) || Object.keys(input).some(key => !allowed.includes(key)) || (input.source === 'local' && (typeof input.endpoint !== 'string' || !input.endpoint.trim() || !['development', 'production'].includes(input.environment)))) throw new AIConnectionContractError('AI_REQUEST_INVALID');
      const response = await options.request<AIConnectionTestResult>(options.path || 'admin/ai-settings/test-connection', { method: 'POST', body: input });
      return validateAIConnectionTestResult(response?.data, input);
    },
  };
}
