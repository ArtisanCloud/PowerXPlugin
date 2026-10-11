package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"strings"
)

const DocumentIngestionSchema = "powerx.knowledge.document-ingestion/v1"

type DocumentRebuildMode string

const (
	RebuildReuseSnapshot DocumentRebuildMode = "reuse_snapshot"
	RebuildApplyConfig   DocumentRebuildMode = "apply_config"
)

// DocumentRebuildSettings follows Core's explicit Host rebuild contract.
// ReuseSnapshot never reads latest profiles; ApplyConfig requires an explicit
// typed configuration. Core owns task deduplication and atomic index switching.
type DocumentRebuildSettings struct {
	Indexing       *SemanticIndexingSettings  `json:"indexing,omitempty"`
	Mode           DocumentRebuildMode        `json:"mode"`
	IdempotencyKey string                     `json:"idempotency_key"`
	Ingestion      *DocumentIngestionSettings `json:"ingestion,omitempty"`
}

func (in *DocumentRebuildSettings) Validate() error {
	if in == nil || strings.TrimSpace(in.IdempotencyKey) == "" || strings.TrimSpace(in.IdempotencyKey) != in.IdempotencyKey || len(in.IdempotencyKey) > 128 {
		return NewError(CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
	}
	switch in.Mode {
	case RebuildReuseSnapshot:
		if in.Ingestion != nil || in.Indexing != nil {
			return NewError(CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
		}
	case RebuildApplyConfig:
		if in.Ingestion == nil && in.Indexing == nil {
			return NewError(CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
		}
		if err := in.Indexing.Validate(); err != nil {
			return err
		}
		return in.Ingestion.Validate()
	default:
		return NewError(CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument")
	}
	return nil
}

type DocumentJobConfig struct {
	DocumentUUID    string                    `json:"document_uuid"`
	EffectiveConfig DocumentIngestionSnapshot `json:"effective_config"`
}

// These are the published Host DTOs, distinct from Local IngestionConfig.
type DocumentProfileRef struct {
	UUID    string `json:"uuid"`
	Version int    `json:"version"`
}
type DocumentStrategyRef struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Version int    `json:"version"`
}

// 指针区分未提供与显式零/false，切块长度单位为 Unicode rune。
type DocumentIngestionSettings struct {
	Schema              string               `json:"schema"`
	Strategy            *DocumentStrategyRef `json:"strategy,omitempty"`
	IngestionProfile    *DocumentProfileRef  `json:"ingestion_profile,omitempty"`
	ProcessorProfile    *DocumentProfileRef  `json:"processor_profile,omitempty"`
	MaskingProfile      *DocumentProfileRef  `json:"masking_profile,omitempty"`
	Priority            *string              `json:"priority,omitempty"`
	SegmentMode         *string              `json:"segment_mode,omitempty"`
	SegmentSizePolicy   *string              `json:"segment_size_policy,omitempty"`
	ChunkSize           *int                 `json:"chunk_size,omitempty"`
	ChunkOverlap        *int                 `json:"chunk_overlap,omitempty"`
	SegmentOrder        *[]string            `json:"segment_order,omitempty"`
	Separators          *[]string            `json:"separators,omitempty"`
	PagePriority        *bool                `json:"page_priority,omitempty"`
	AnchorHeadingPath   *bool                `json:"anchor_heading_path,omitempty"`
	AnchorClauseID      *bool                `json:"anchor_clause_id,omitempty"`
	AnchorRowNumber     *bool                `json:"anchor_row_number,omitempty"`
	AnchorSpeaker       *bool                `json:"anchor_speaker,omitempty"`
	AnchorSentenceIndex *bool                `json:"anchor_sentence_index,omitempty"`
}
type DocumentFrozenProfile struct {
	DocumentProfileRef
	Key            string `json:"key"`
	ConfigChecksum string `json:"config_checksum"`
}
type DocumentIngestionSnapshot struct {
	Indexing            *SemanticIndexSnapshot `json:"indexing,omitempty"`
	Schema              string                 `json:"schema"`
	IndexMode           string                 `json:"index_mode"`
	LengthUnit          string                 `json:"length_unit"`
	Strategy            *DocumentStrategyRef   `json:"strategy,omitempty"`
	IngestionProfile    *DocumentFrozenProfile `json:"ingestion_profile,omitempty"`
	IndexProfile        *DocumentFrozenProfile `json:"index_profile,omitempty"`
	RAGProfile          *DocumentFrozenProfile `json:"rag_profile,omitempty"`
	Priority            string                 `json:"priority"`
	SegmentMode         string                 `json:"segment_mode"`
	SegmentSizePolicy   string                 `json:"segment_size_policy"`
	ChunkSize           int                    `json:"chunk_size"`
	ChunkOverlap        int                    `json:"chunk_overlap"`
	SegmentOrder        []string               `json:"segment_order"`
	Separators          []string               `json:"separators"`
	PagePriority        bool                   `json:"page_priority"`
	AnchorHeadingPath   bool                   `json:"anchor_heading_path"`
	AnchorClauseID      bool                   `json:"anchor_clause_id"`
	AnchorRowNumber     bool                   `json:"anchor_row_number"`
	AnchorSpeaker       bool                   `json:"anchor_speaker"`
	AnchorSentenceIndex bool                   `json:"anchor_sentence_index"`
	SourceChecksum      string                 `json:"source_checksum"`
	SourceVersion       string                 `json:"source_version"`
}

// Validate checks references and basic bounds; Core validates tenant ownership,
// frozen Profile settings, content-specific modes and execution constraints.
func (in *DocumentIngestionSettings) Validate() error {
	invalid := func() error { return NewError(CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument") }
	if in == nil {
		return nil
	}
	if in.Schema != DocumentIngestionSchema {
		return invalid()
	}
	for _, profile := range []*DocumentProfileRef{in.IngestionProfile, in.ProcessorProfile, in.MaskingProfile} {
		if profile != nil {
			id, err := uuid.Parse(profile.UUID)
			if err != nil || id == uuid.Nil || id.String() != profile.UUID || profile.Version < 1 {
				return invalid()
			}
		}
	}
	if in.Strategy != nil {
		if in.Strategy.Version < 1 || (in.Strategy.UUID == "" && in.Strategy.Key == "") {
			return invalid()
		}
		if in.Strategy.UUID != "" {
			id, err := uuid.Parse(in.Strategy.UUID)
			if err != nil || id == uuid.Nil || id.String() != in.Strategy.UUID {
				return invalid()
			}
		}
	}
	if in.ChunkSize != nil && (*in.ChunkSize < 0 || *in.ChunkSize > 65536) {
		return invalid()
	}
	if in.ChunkOverlap != nil && *in.ChunkOverlap < 0 {
		return invalid()
	}
	if in.ChunkSize != nil && in.ChunkOverlap != nil && ((*in.ChunkSize == 0 && *in.ChunkOverlap != 0) || (*in.ChunkSize > 0 && *in.ChunkOverlap >= *in.ChunkSize)) {
		return invalid()
	}
	return nil
}

type DocumentIngestionCapabilities struct {
	Schema                    string                    `json:"schema"`
	IndexMode                 string                    `json:"index_mode"`
	ContentTypes              []string                  `json:"content_types"`
	LengthUnit                string                    `json:"length_unit"`
	ChunkSizeMin              int                       `json:"chunk_size_min"`
	ChunkSizeMax              int                       `json:"chunk_size_max"`
	Priorities                []string                  `json:"priorities"`
	PrioritySemantics         string                    `json:"priority_semantics"`
	SegmentModes              []string                  `json:"segment_modes"`
	SegmentSizePolicies       []string                  `json:"segment_size_policies"`
	SegmentOrderItems         []string                  `json:"segment_order_items"`
	ProcessorProfileSupported bool                      `json:"processor_profile_supported"`
	MaskingProfileSupported   bool                      `json:"masking_profile_supported"`
	PagePrioritySupported     bool                      `json:"page_priority_supported"`
	StrategyOverrideSupported bool                      `json:"strategy_override_supported"`
	AnchorModeRequirements    map[string]string         `json:"anchor_mode_requirements"`
	Defaults                  DocumentIngestionSettings `json:"defaults"`
	MaxContentChunks          int                       `json:"max_content_chunks"`
	ProfilePrecedence         []string                  `json:"profile_precedence"`
}

type IndexJobChunk struct {
	UUID         string          `json:"uuid"`
	SpaceUUID    string          `json:"space_uuid"`
	JobUUID      string          `json:"job_uuid"`
	DocumentUUID string          `json:"document_uuid"`
	Ordinal      int             `json:"ordinal"`
	Kind         string          `json:"kind"`
	Content      string          `json:"content"`
	Checksum     string          `json:"checksum"`
	Metadata     json.RawMessage `json:"metadata"`
}

type IndexJobChunksReader interface {
	GetIndexJobChunks(context.Context, IndexJobQuery) ([]IndexJobChunk, error)
}

// ValidateHostDocument preserves the exact UTF-8 source used by Core checksums.
func ValidateHostDocument(document KnowledgeDocument) (KnowledgeDocument, error) {
	source, version := document.Content, document.Version
	document = document.Normalized()
	document.Content, document.Version = source, version
	if err := document.Validate(); err != nil {
		return KnowledgeDocument{}, err
	}
	if document.Checksum == "" {
		sum := sha256.Sum256([]byte(source))
		document.Checksum = hex.EncodeToString(sum[:])
	}
	if document.Version == "" {
		document.Version = document.Checksum
	}
	document.Metadata = RedactMap(document.Metadata)
	return document, nil
}
