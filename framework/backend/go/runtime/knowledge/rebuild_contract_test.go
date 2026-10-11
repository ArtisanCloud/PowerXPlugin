package knowledge

import "testing"

func TestDocumentRebuildModeContract(t *testing.T) {
	config := &DocumentIngestionSettings{Schema: DocumentIngestionSchema}
	for _, tc := range []struct {
		name    string
		request *DocumentRebuildSettings
		valid   bool
	}{
		{"reuse", &DocumentRebuildSettings{Mode: RebuildReuseSnapshot, IdempotencyKey: "request-1"}, true},
		{"apply", &DocumentRebuildSettings{Mode: RebuildApplyConfig, IdempotencyKey: "request-2", Ingestion: config}, true},
		{"reuse_with_override", &DocumentRebuildSettings{Mode: RebuildReuseSnapshot, IdempotencyKey: "request-1", Ingestion: config}, false},
		{"apply_without_config", &DocumentRebuildSettings{Mode: RebuildApplyConfig, IdempotencyKey: "request-2"}, false},
		{"missing_key", &DocumentRebuildSettings{Mode: RebuildReuseSnapshot}, false},
		{"unknown_mode", &DocumentRebuildSettings{Mode: "latest", IdempotencyKey: "request-3"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.request.Validate(); (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}
