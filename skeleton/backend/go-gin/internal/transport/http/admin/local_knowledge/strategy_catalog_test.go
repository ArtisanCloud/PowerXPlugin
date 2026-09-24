package local_knowledge

import (
	"encoding/json"
	"testing"
)

func TestLocalStrategyCatalogMatchesCorePackageSet(t *testing.T) {
	catalog, err := loadStrategyCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if got, want := len(catalog.StrategyPackages), 18; got != want {
		t.Fatalf("strategy package count=%d want=%d", got, want)
	}
	if got, want := catalog.StrategyPackages["H_fusion"].ProfileKey, "p1_general"; got != want {
		t.Fatalf("H_fusion profile=%q want=%q", got, want)
	}
	if got, want := catalog.StrategyPackages["H_fusion"].Dependencies.Index, []string{"index.dense", "index.sparse"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("H_fusion dependencies=%v want=%v", got, want)
	}
	ordered := orderedStrategies(catalog)
	if len(ordered) != 18 || ordered[0].Key != "A_simple" || ordered[4].Key != "H_fusion" {
		t.Fatalf("unexpected Core package order: %#v", ordered)
	}
	if ready, missing := localStrategyReadiness(catalog.StrategyPackages["H_fusion"], catalog.Bundles["p1_general"], true, false); !ready || len(missing) != 0 {
		t.Fatalf("Fusion should be available once the local embedding executor is configured: ready=%v missing=%v", ready, missing)
	}
	if ready, _ := localStrategyReadiness(catalog.StrategyPackages["H_fusion"], catalog.Bundles["p1_general"], false, false); ready {
		t.Fatal("Fusion must remain unavailable without a local embedding executor")
	}
	if ready, missing := localStrategyReadiness(catalog.StrategyPackages["I_hyde"], catalog.Bundles["p1_general"], true, true); !ready || len(missing) != 0 {
		t.Fatalf("HyDE should require and accept both local dense and LLM drivers: ready=%v missing=%v", ready, missing)
	}
	if ready, _ := localStrategyReadiness(catalog.StrategyPackages["I_hyde"], catalog.Bundles["p1_general"], true, false); ready {
		t.Fatal("HyDE must remain unavailable without a configured local LLM")
	}
	for _, key := range []string{"F_rerank", "G_rse", "K_kg", "N_self_rag", "O_crag"} {
		item := catalog.StrategyPackages[key]
		if ready, missing := localStrategyReadiness(item, catalog.Bundles[item.ProfileKey], true, true); !ready || len(missing) != 0 {
			t.Fatalf("%s should be available with explicit local dense and LLM executors: ready=%v missing=%v", key, ready, missing)
		}
		if ready, _ := localStrategyReadiness(item, catalog.Bundles[item.ProfileKey], true, false); ready {
			t.Fatalf("%s must remain unavailable without a configured local LLM", key)
		}
	}
}

func TestBuildKnowledgeChunksPreservesMarkdownSectionsForHierarchicalStrategies(t *testing.T) {
	parts := buildKnowledgeChunks("# 退款政策\n支持七天退款。\n# 配送说明\n支持次日配送。", "J_hier", 100, 10)
	if len(parts) != 2 {
		t.Fatalf("chunk count=%d want 2", len(parts))
	}
	var metadata struct {
		Section string `json:"section"`
	}
	if err := json.Unmarshal(parts[0].Metadata, &metadata); err != nil || metadata.Section != "退款政策" {
		t.Fatalf("first chunk metadata=%s section=%q err=%v", parts[0].Metadata, metadata.Section, err)
	}
	if err := json.Unmarshal(parts[1].Metadata, &metadata); err != nil || metadata.Section != "配送说明" {
		t.Fatalf("second chunk metadata=%s section=%q err=%v", parts[1].Metadata, metadata.Section, err)
	}
}

func TestDocumentAugmentationAddsInspectableStructuredFields(t *testing.T) {
	parts := augmentKnowledgeChunks([]knowledgeChunkPart{{Content: "退款政策支持七天无理由退款", Metadata: []byte(`{}`)}}, "退款政策", "D_doc_augmentation")
	if len(parts) != 1 {
		t.Fatalf("parts=%d want 1", len(parts))
	}
	var metadata struct {
		Title    string   `json:"title"`
		Keywords []string `json:"augmented_keywords"`
	}
	if err := json.Unmarshal(parts[0].Metadata, &metadata); err != nil || metadata.Title != "退款政策" || len(metadata.Keywords) == 0 {
		t.Fatalf("augmentation metadata=%s title=%q keywords=%v err=%v", parts[0].Metadata, metadata.Title, metadata.Keywords, err)
	}
}
