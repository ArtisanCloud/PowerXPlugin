package knowledge

import (
	"github.com/google/uuid"
	"strings"
)

// These DTOs follow Core's Knowledge Host provisioning service contract.
// Tenant and actor are deliberately absent: the trusted service credential owns both.
type SpaceQuotas struct {
	CPUCores             int `json:"cpu_cores"`
	StorageGB            int `json:"storage_gb"`
	IngestionConcurrency int `json:"ingestion_concurrency"`
}
type ProfileRef struct {
	UUID    string `json:"uuid"`
	Key     string `json:"key"`
	Version int    `json:"version"`
}
type ProfileMapping struct {
	Ingestion *ProfileRef `json:"ingestion,omitempty"`
	Index     *ProfileRef `json:"index,omitempty"`
	RAG       *ProfileRef `json:"rag,omitempty"`
}
type PolicyTemplateRef struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version string `json:"version"`
}
type CreateSpaceInput struct {
	Name                 string       `json:"name"`
	DepartmentUUID       string       `json:"department_uuid"`
	StrategyKey          string       `json:"strategy_key"`
	SceneKey             string       `json:"scene_key,omitempty"`
	PolicyTemplateUUID   string       `json:"policy_template_uuid,omitempty"`
	IngestionProfileUUID string       `json:"ingestion_profile_uuid,omitempty"`
	IndexProfileUUID     string       `json:"index_profile_uuid,omitempty"`
	RAGProfileUUID       string       `json:"rag_profile_uuid,omitempty"`
	Quotas               *SpaceQuotas `json:"quotas,omitempty"`
}
type CreatedSpace struct {
	SpaceUUID          string         `json:"space_uuid"`
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	DepartmentUUID     string         `json:"department_uuid"`
	StrategyKey        string         `json:"strategy_key"`
	SceneKey           string         `json:"scene_key"`
	PolicyTemplateUUID string         `json:"policy_template_uuid"`
	Profiles           ProfileMapping `json:"profiles"`
	Quotas             SpaceQuotas    `json:"quotas"`
}

func validProvisioningUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}
func (in CreateSpaceInput) Validate() error {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 128 || strings.TrimSpace(in.StrategyKey) == "" || !validProvisioningUUID(in.DepartmentUUID) {
		return NewError(CodeInvalidArgument, "knowledge.provisioning.invalid_argument")
	}
	for _, id := range []string{in.PolicyTemplateUUID, in.IngestionProfileUUID, in.IndexProfileUUID, in.RAGProfileUUID} {
		if id != "" && !validProvisioningUUID(id) {
			return NewError(CodeInvalidArgument, "knowledge.provisioning.invalid_uuid")
		}
	}
	if in.Quotas != nil && (in.Quotas.CPUCores < 1 || in.Quotas.StorageGB < 50 || in.Quotas.IngestionConcurrency < 1) {
		return NewError(CodeInvalidArgument, "knowledge.provisioning.invalid_quotas")
	}
	return nil
}

// ValidateCreatedSpace rejects a malformed success; it never invents missing UUIDs.
func ValidateCreatedSpace(item *CreatedSpace) error {
	if item == nil || !validProvisioningUUID(item.SpaceUUID) || !validProvisioningUUID(item.DepartmentUUID) || !validProvisioningUUID(item.PolicyTemplateUUID) || strings.TrimSpace(item.Name) == "" || item.Status != "pending_iam" || item.StrategyKey == "" || item.SceneKey == "" {
		return NewError(CodeInvalidResponse, "knowledge.provisioning.invalid_response")
	}
	for _, profile := range []*ProfileRef{item.Profiles.Ingestion, item.Profiles.Index, item.Profiles.RAG} {
		if profile == nil || !validProvisioningUUID(profile.UUID) || profile.Key == "" || profile.Version < 1 {
			return NewError(CodeInvalidResponse, "knowledge.provisioning.invalid_response")
		}
	}
	if item.Quotas.CPUCores < 1 || item.Quotas.StorageGB < 50 || item.Quotas.IngestionConcurrency < 1 {
		return NewError(CodeInvalidResponse, "knowledge.provisioning.invalid_response")
	}
	return nil
}

func ValidateHostCatalog(catalog *KnowledgeCatalog) error {
	invalid := func() error { return NewError(CodeInvalidResponse, "knowledge.provisioning.invalid_catalog") }
	if catalog == nil || catalog.Source != "powerx_core" || catalog.Version == "" || catalog.Scenes == nil || catalog.StrategyPackages == nil || catalog.PolicyTemplates == nil || catalog.QuotaDefaults.CPUCores < 1 || catalog.QuotaDefaults.StorageGB < 50 || catalog.QuotaDefaults.IngestionConcurrency < 1 || catalog.QuotaMinimums.CPUCores < 1 || catalog.QuotaMinimums.StorageGB < 50 || catalog.QuotaMinimums.IngestionConcurrency < 1 {
		return invalid()
	}
	foundDefault := catalog.DefaultPolicyTemplateUUID == ""
	for _, template := range catalog.PolicyTemplates {
		if !validProvisioningUUID(template.UUID) || template.Name == "" || template.Version == "" {
			return invalid()
		}
		if template.UUID == catalog.DefaultPolicyTemplateUUID {
			foundDefault = true
		}
	}
	if !foundDefault {
		return invalid()
	}
	for _, strategy := range catalog.StrategyPackages {
		if strategy.Key == "" || strategy.Label == "" {
			return invalid()
		}
		for _, profile := range []*ProfileRef{strategy.Profiles.Ingestion, strategy.Profiles.Index, strategy.Profiles.RAG} {
			if profile == nil {
				if strategy.Available {
					return invalid()
				}
				continue
			}
			if !validProvisioningUUID(profile.UUID) || profile.Key == "" || profile.Version < 1 {
				return invalid()
			}
		}
	}
	return nil
}
