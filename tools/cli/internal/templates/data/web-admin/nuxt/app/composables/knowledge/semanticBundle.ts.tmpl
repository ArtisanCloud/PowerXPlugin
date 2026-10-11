import type {SemanticDocument,SemanticIndexing,SemanticArtifact,SourceRef} from '../api/knowledgeSemanticTypes'
export type SemanticDraft={externalUUID:string;sourceArtifactUUID:string;profileUUID:string;profileArtifactUUID:string;key:string;intent?:string}
export function newSemanticDraft():SemanticDraft{return {externalUUID:crypto.randomUUID(),sourceArtifactUUID:crypto.randomUUID(),profileUUID:crypto.randomUUID(),profileArtifactUUID:crypto.randomUUID(),key:crypto.randomUUID()}}
export async function utf8Checksum(text:string):Promise<string>{const digest=await crypto.subtle.digest('SHA-256',new TextEncoder().encode(text));return Array.from(new Uint8Array(digest)).map(value=>value.toString(16).padStart(2,'0')).join('')}
// Only builds explicit test bundle data; it performs no business profile extraction.
export async function semanticBundle(draft:SemanticDraft,text:string,indexing:SemanticIndexing,profileText:string,ingestion:unknown,kind:'raw_document'|'cleaned_document',title='',categoryCodes:string[]=[],tagUUIDs:string[]=[]):Promise<SemanticDocument>{
 const sourceHash=await utf8Checksum(text);const version=sourceHash
 const source_ref:SourceRef={kind,source_uuid:draft.externalUUID,version,checksum:sourceHash,position_unit:'unicode_codepoint',char_start:0,char_end:Array.from(text).length}
 const artifacts:SemanticArtifact[]=[{uuid:draft.sourceArtifactUUID,role:'source_chunk',text,checksum:sourceHash,version,source_ref,category_codes:[...categoryCodes],tag_uuids:[...tagUUIDs]}]
 if(profileText.trim()){
  const checksum=await utf8Checksum(profileText)
  artifacts.push({uuid:draft.profileArtifactUUID,role:'knowledge_profile',text:profileText,checksum,version:checksum,knowledge_profile_uuid:draft.profileUUID,source_ref:{kind:'knowledge_profile',source_uuid:draft.profileUUID,version:checksum,checksum,position_unit:'unicode_codepoint',char_start:0,char_end:Array.from(profileText).length},category_codes:[...categoryCodes],tag_uuids:[...tagUUIDs]})
 }
 const effective={...indexing,artifact_roles:artifacts.map(item=>item.role)}
 const intent=JSON.stringify({title,artifacts,indexing:effective,ingestion})
 if(draft.intent && draft.intent!==intent)draft.key=crypto.randomUUID()
 draft.intent=intent
 return {indexing:effective,artifacts,external_ref:{document_uuid:draft.externalUUID},idempotency_key:draft.key,uri:'plugin-knowledge://'+draft.externalUUID,version,checksum:sourceHash}
}
