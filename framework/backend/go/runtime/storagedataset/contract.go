// Package storagedataset runs Core-authorized dataset operations in a plugin.
package storagedataset

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	HostVersion         = "powerx.storage-dataset-host/v1"
	ExecutionVersion    = "powerx.storage-dataset-execution/v1"
	RegisterCapability  = "com.corex.storage.datasets.register"
	PreviewCapability   = "com.corex.storage.datasets.preview_worker"
	ExecutionCapability = "com.corex.storage.datasets.execution_worker"
	ConsumeCapability   = "com.corex.runtime.taskqueue.service_consume"
	PreviewSubscriber   = "powerx.storage.dataset-preview.v1"
	ExecutionSubscriber = "powerx.storage.dataset-execution.v1"
	PreviewTopic        = "storage.dataset.preview.requested"
	ExecutionTopic      = "storage.dataset.execution.requested"
	MaxExamined         = 2048
	MaxAffected         = 128
	MaxArchiveBytes     = 256 << 10
)

var (
	ErrContract  = errors.New("DATASET_CONTRACT_INVALID")
	ErrScope     = errors.New("DATASET_SCOPE_MISMATCH")
	ErrDeadline  = errors.New("DATASET_DEADLINE_EXCEEDED")
	ErrSchema    = errors.New("DATASET_SCHEMA_MISMATCH")
	ErrSource    = errors.New("DATASET_SOURCE_UNAVAILABLE")
	ErrAdapter   = errors.New("DATASET_ADAPTER_UNAVAILABLE")
	ErrReconcile = errors.New("DATASET_SOURCE_RECONCILIATION_REQUIRED")
	ErrReceipt   = errors.New("DATASET_SOURCE_RECEIPT_CONFLICT")
)

type Scope struct {
	RequestUUID        string    `json:"request_uuid"`
	DatasetUUID        string    `json:"dataset_uuid"`
	TenantUUID         string    `json:"tenant_uuid"`
	SourceBindingUUID  string    `json:"source_binding_uuid"`
	SourceFingerprint  string    `json:"source_fingerprint"`
	AcceptanceRevision int64     `json:"acceptance_revision"`
	DefinitionDigest   string    `json:"definition_digest"`
	SchemaVersion      int64     `json:"schema_contract_version"`
	AdapterKey         string    `json:"adapter_key"`
	AdapterVersion     int64     `json:"adapter_version"`
	Cutoff             time.Time `json:"cutoff_at"`
	Deadline           time.Time `json:"deadline_at"`
	MaxExamined        int       `json:"max_examined"`
}

type Grant struct {
	Scope
	GrantUUID      string    `json:"grant_uuid"`
	Action         string    `json:"action"`
	PolicyRevision int64     `json:"policy_revision"`
	IssuedAt       time.Time `json:"issued_at"`
	MaxAffected    int       `json:"max_affected"`
}

type Claim struct {
	RequestUUID string `json:"request_uuid"`
	ReceiptUUID string `json:"receipt_uuid"`
	Fence       uint64 `json:"fence"`
}

type ExecutionClaim struct {
	State        string `json:"state"`
	Grant        *Grant `json:"grant,omitempty"`
	ExpiredGrant *Grant `json:"expired_grant,omitempty"`
	RequestUUID  string `json:"request_uuid,omitempty"`
	GrantUUID    string `json:"grant_uuid,omitempty"`
}

type Registration struct {
	RequestUUID        string `json:"request_uuid"`
	DatasetUUID        string `json:"dataset_uuid"`
	AcceptanceRevision int64  `json:"acceptance_revision"`
	State              string `json:"registration_state"`
	Reason             string `json:"status_reason,omitempty"`
}

type ProtectionCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type Counts struct {
	Coverage   string            `json:"coverage"`
	Examined   int               `json:"examined_count"`
	Candidates int               `json:"candidate_count"`
	Protected  []ProtectionCount `json:"protected_by_reason"`
}

type PreviewReport struct {
	Claim
	AcceptanceRevision int64   `json:"acceptance_revision"`
	Outcome            string  `json:"outcome"`
	Result             *Counts `json:"result,omitempty"`
	Reason             string  `json:"status_reason,omitempty"`
}

type Report struct {
	RequestUUID    string    `json:"request_uuid"`
	GrantUUID      string    `json:"grant_uuid"`
	ReceiptUUID    string    `json:"source_receipt_uuid"`
	Digest         string    `json:"source_receipt_digest"`
	Outcome        string    `json:"outcome"`
	Affected       int       `json:"affected_count"`
	SourceSealedAt time.Time `json:"source_sealed_at"`
}

type Result struct {
	RequestUUID string `json:"request_uuid"`
	State       string `json:"state"`
}

//go:embed contracts/*.json
var schemas embed.FS

type schemaKey struct{ version, definition string }
type compiledSchema struct {
	schema *jsonschema.Schema
	err    error
}

var schemaCache sync.Map

func contractSchema(version, definition string) (*jsonschema.Schema, error) {
	key := schemaKey{version, definition}
	if cached, ok := schemaCache.Load(key); ok {
		entry := cached.(compiledSchema)
		return entry.schema, entry.err
	}
	file := "plugin-dataset-host.v1.schema.json"
	if version == ExecutionVersion {
		file = "plugin-dataset-execution.v1.schema.json"
	}
	raw, err := schemas.ReadFile("contracts/" + file)
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true
	const uri = "https://powerx.invalid/storage-dataset.json"
	if err := compiler.AddResource(uri, bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	ref := uri
	if definition != "" {
		ref += "#/$defs/" + definition
	}
	schema, err := compiler.Compile(ref)
	schemaCache.Store(key, compiledSchema{schema, err})
	return schema, err
}

func decodeContract(raw []byte, version, definition string, out any) error {
	if len(raw) > 400<<10 || !utf8.Valid(raw) {
		return ErrContract
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	value, err := readJSON(dec, 0)
	if err != nil {
		return ErrContract
	}
	if _, err := dec.Token(); err != io.EOF {
		return ErrContract
	}
	schema, err := contractSchema(version, definition)
	if err != nil {
		return fmt.Errorf("%w: schema unavailable", ErrContract)
	}
	if schema.Validate(value) != nil {
		// Schema diagnostics may include data from the wire, including archive bytes.
		return ErrContract
	}
	if out != nil && json.Unmarshal(raw, out) != nil {
		return ErrContract
	}
	return nil
}

// readJSON preserves numeric precision and rejects ambiguous object keys.
func readJSON(dec *json.Decoder, level int) (any, error) {
	if level > 48 {
		return nil, ErrContract
	}
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch token {
	case json.Delim('{'):
		value := make(map[string]any)
		for dec.More() {
			key, err := dec.Token()
			name, ok := key.(string)
			if err != nil || !ok {
				return nil, ErrContract
			}
			if _, exists := value[name]; exists {
				return nil, ErrContract
			}
			item, err := readJSON(dec, level+1)
			if err != nil {
				return nil, err
			}
			value[name] = item
		}
		_, err = dec.Token()
		return value, err
	case json.Delim('['):
		value := make([]any, 0)
		for dec.More() {
			item, err := readJSON(dec, level+1)
			if err != nil {
				return nil, err
			}
			value = append(value, item)
		}
		_, err = dec.Token()
		return value, err
	default:
		return token, nil
	}
}
