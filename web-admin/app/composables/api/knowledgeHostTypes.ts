import type {SemanticIndexing,SemanticIndexSnapshot} from './knowledgeSemanticTypes'
export const documentIngestionSchema = 'powerx.knowledge.document-ingestion/v1'
export type HostProfileRef = { uuid: string; version: number; key?: string; config_checksum?: string }
export type HostIngestionSettings = {
 schema: string; strategy?: { uuid?: string; key?: string; version: number }; ingestion_profile?: HostProfileRef
 priority?: string; segment_mode?: string; segment_size_policy?: string; chunk_size?: number; chunk_overlap?: number
 segment_order?: string[]; separators?: string[]; page_priority?: boolean
 anchor_heading_path?: boolean; anchor_clause_id?: boolean; anchor_row_number?: boolean; anchor_speaker?: boolean; anchor_sentence_index?: boolean
}
export type HostIngestionSnapshot = HostIngestionSettings & { index_mode: string; indexing?: SemanticIndexSnapshot; length_unit: string; index_profile?: HostProfileRef; rag_profile?: HostProfileRef; source_checksum: string; source_version: string }
export type HostIngestionCapabilities = {
 schema: string; index_mode: string; content_types: string[]; length_unit: string; chunk_size_min: number; chunk_size_max: number
 priorities: string[]; segment_modes: string[]; segment_size_policies: string[]; segment_order_items: string[]
 processor_profile_supported: boolean; masking_profile_supported: boolean; page_priority_supported: boolean; strategy_override_supported: boolean
 anchor_mode_requirements: Record<string,string>; defaults: HostIngestionSettings; max_content_chunks: number
}
export type HostIndexJob = { artifact_count?:number;vector_count?:number;bundle_generation?:string;invocation_trace_id?:string;request_id?:string; rebuild_mode?: 'reuse_snapshot' | 'apply_config'; document_count?: number; processed_documents?: number; document_configs?: {document_uuid: string; effective_config: HostIngestionSnapshot}[]; job_id: string; space_id?: string; document_id?: string; status: string; error_code?: string; priority?: string; trace_id?: string; effective_config?: HostIngestionSnapshot; requested_config?: HostIngestionSettings; snapshot_checksum?: string; chunk_count?: number; started_at?: string; finished_at?: string }
export type HostIndexJobChunk = { uuid: string; job_uuid: string; space_uuid: string; document_uuid: string; ordinal: number; kind: string; content: string; checksum: string; metadata: Record<string,unknown> }

export type HostRebuildRequest = { mode: 'reuse_snapshot'; idempotency_key: string; ingestion?: never } | { mode: 'apply_config'; idempotency_key: string; ingestion?: HostIngestionSettings; indexing?:SemanticIndexing }
