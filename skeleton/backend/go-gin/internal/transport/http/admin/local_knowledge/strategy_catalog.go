package local_knowledge

import (
	_ "embed"
	"gopkg.in/yaml.v3"
)

//go:embed strategy_catalog.yaml
var strategyCatalogYAML []byte

type strategyCatalog struct {
	StrategyPackages map[string]strategyPackage `yaml:"strategy_packages"`
	Bundles          map[string]strategyBundle  `yaml:"strategy_bundles"`
}
type strategyPackage struct {
	Key          string `json:"key" yaml:"-"`
	ProfileKey   string `json:"profile_key" yaml:"recommended_profile_key"`
	Dependencies struct {
		Index   []string `json:"index" yaml:"index"`
		Runtime []string `json:"runtime" yaml:"runtime"`
		Assets  []string `json:"assets" yaml:"assets"`
	} `json:"dependencies" yaml:"dependencies"`
}
type strategyBundle struct {
	Prerequisites []string `yaml:"prerequisites"`
}

var strategyPackageOrder = []string{"A_simple", "A0_acl", "C_context_enriched", "F_rerank", "H_fusion", "A1_routing", "E_query_transform", "G_rse", "I_hyde", "L_feedback", "M_adaptive", "N_self_rag", "B_semantic_chunking", "J_hier", "D_doc_augmentation", "A2_time_aware", "K_kg", "O_crag"}

func loadStrategyCatalog() (strategyCatalog, error) {
	var catalog strategyCatalog
	if err := yaml.Unmarshal(strategyCatalogYAML, &catalog); err != nil {
		return strategyCatalog{}, err
	}
	for key, item := range catalog.StrategyPackages {
		item.Key = key
		catalog.StrategyPackages[key] = item
	}
	return catalog, nil
}
func orderedStrategies(catalog strategyCatalog) []strategyPackage {
	items := make([]strategyPackage, 0, len(catalog.StrategyPackages))
	for _, key := range strategyPackageOrder {
		if item, ok := catalog.StrategyPackages[key]; ok {
			items = append(items, item)
		}
	}
	return items
}
func findStrategyPackage(catalog strategyCatalog, key string) (strategyPackage, bool) {
	item, ok := catalog.StrategyPackages[key]
	return item, ok
}

// A catalog prerequisite is never mistaken for an available local executor.
// In particular, A0 requires a document-level ACL enforcer and normalized
// metadata. Tenant scoping alone is not an ACL implementation.
func localStrategyReadiness(item strategyPackage, bundle strategyBundle, denseAvailable, llmAvailable bool) (bool, []string) {
	if item.Key == "A1_routing" {
		return true, nil
	}
	if item.Key == "L_feedback" && denseAvailable {
		return true, nil
	}
	if item.Key == "M_adaptive" && denseAvailable {
		return true, nil
	}
	// The standalone adapter has no hidden rerank fallback.  F is available
	// only when an explicitly configured local LLM can perform the structured
	// candidate-ranking call used by this adapter.
	if item.Key == "F_rerank" && denseAvailable && llmAvailable {
		return true, nil
	}
	if item.Key == "G_rse" && denseAvailable && llmAvailable {
		return true, nil
	}
	if item.Key == "K_kg" && denseAvailable && llmAvailable {
		return true, nil
	}
	if (item.Key == "N_self_rag" || item.Key == "O_crag") && denseAvailable && llmAvailable {
		return true, nil
	}
	// The local retrieval adapter provides a BM25 sparse scorer without an
	// external service. Dense and Fusion additionally require a real local
	// embedding model, because their document index and query vector must use
	// the same model key.
	if (item.Key == "A_simple" || item.Key == "A0_acl" || item.Key == "A2_time_aware" || item.Key == "B_semantic_chunking" || item.Key == "C_context_enriched" || item.Key == "D_doc_augmentation" || item.Key == "H_fusion" || item.Key == "J_hier") && denseAvailable {
		return true, nil
	}
	// HyDE is a real two-stage local pipeline: a configured local LLM generates
	// the hypothetical passage and the configured local embedding model encodes
	// it for the same persisted dense index. It is not enabled when either
	// driver is absent.
	if (item.Key == "E_query_transform" || item.Key == "I_hyde") && denseAvailable && llmAvailable {
		return true, nil
	}
	missing := append([]string{}, item.Dependencies.Index...)
	missing = append(missing, item.Dependencies.Runtime...)
	missing = append(missing, item.Dependencies.Assets...)
	missing = append(missing, bundle.Prerequisites...)
	if len(missing) == 0 {
		missing = []string{"runtime." + item.Key}
	}
	return false, missing
}
