import type { LocalKnowledgeDocument } from '../api/useLocalKnowledge'
import type { RetrievalMode } from '../api/knowledgeSemanticTypes'

// UI view model; a Core wire contract must be published before a Host adapter is enabled.
export type KnowledgeContentDocument = {
  uuid: string
  title: string
  preview: string
  suggested_questions: string[]
}
export type KnowledgeContentOverview = {
  space_uuid: string
  scope: 'space_documents' | 'queryable_documents'
  summary?: string
  topics: string[]
  documents: KnowledgeContentDocument[]
  total: number
  next_cursor?: string
  corpus_generation?: string
}
export type KnowledgeContentRequest = {
  mode?: RetrievalMode
  corpus_generation?: string
  cursor?: string
}
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
function invalid(): never {
  throw { data: { code: 'KNOWLEDGE_INVALID_RESPONSE' } }
}
export function localContentOverview(
  spaceUUID: string,
  documents: LocalKnowledgeDocument[]
): KnowledgeContentOverview {
  if (!uuidPattern.test(spaceUUID) || !Array.isArray(documents)) invalid()
  const seen = new Set<string>()
  const items = documents.map((document) => {
    if (
      !document ||
      !uuidPattern.test(document.uuid) ||
      document.space_uuid !== spaceUUID ||
      seen.has(document.uuid) ||
      typeof document.title !== 'string' ||
      !document.title.trim() ||
      uuidPattern.test(document.title.trim()) ||
      typeof document.content !== 'string'
    )
      invalid()
    seen.add(document.uuid)
    return {
      uuid: document.uuid,
      title: document.title,
      preview: Array.from(document.content).slice(0, 320).join(''),
      suggested_questions: []
    }
  })
  return {
    space_uuid: spaceUUID,
    scope: 'space_documents',
    topics: [],
    documents: items,
    total: items.length
  }
}
export function validateContentOverview(
  value: KnowledgeContentOverview,
  spaceUUID: string,
  request: KnowledgeContentRequest
) {
  if (
    !value ||
    value.space_uuid !== spaceUUID ||
    !['space_documents', 'queryable_documents'].includes(value.scope) ||
    !Number.isInteger(value.total) ||
    value.total < 0 ||
    !Array.isArray(value.documents) ||
    value.documents.length > value.total ||
    (value.total > 0 && !value.documents.length) ||
    !Array.isArray(value.topics) ||
    value.topics.some(
      (topic) => typeof topic !== 'string' || !topic.trim() || uuidPattern.test(topic.trim())
    ) ||
    (value.summary !== undefined && typeof value.summary !== 'string') ||
    (value.next_cursor !== undefined &&
      (typeof value.next_cursor !== 'string' || !value.next_cursor.trim()))
  )
    invalid()
  if (value.scope === 'queryable_documents' && !uuidPattern.test(value.corpus_generation || ''))
    invalid()
  if (
    request.corpus_generation &&
    (value.scope !== 'queryable_documents' || value.corpus_generation !== request.corpus_generation)
  )
    invalid()
  const seen = new Set<string>()
  for (const document of value.documents) {
    if (
      !document ||
      !uuidPattern.test(document.uuid) ||
      seen.has(document.uuid) ||
      typeof document.title !== 'string' ||
      !document.title.trim() ||
      uuidPattern.test(document.title.trim()) ||
      typeof document.preview !== 'string' ||
      !Array.isArray(document.suggested_questions) ||
      new Set(document.suggested_questions).size !== document.suggested_questions.length ||
      document.suggested_questions.some(
        (question) => typeof question !== 'string' || !question.trim()
      )
    )
      invalid()
    seen.add(document.uuid)
  }
  if (!request.cursor && value.documents.length < value.total && !value.next_cursor) invalid()
  if (!value.total && value.next_cursor) invalid()
  return value
}
