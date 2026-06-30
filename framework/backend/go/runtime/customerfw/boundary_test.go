package customerfw

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageDoesNotDefineSCRMOrIndustryDomainModels(t *testing.T) {
	forbidden := []string{
		"CustomerProfile",
		"CustomerTag",
		"CustomerOwner",
		"FollowUp",
		"Timeline",
		"Lead",
		"Opportunity",
		"Sales",
		"Guardian",
		"Player",
		"Learner",
		"Patient",
		"Fan",
		"Entitlement",
		"Benefit",
		"GrowthLevel",
		"GrowthReport",
		"SCRM",
	}

	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || path == "doc.go" || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(raw)
		for _, term := range forbidden {
			if strings.Contains(content, term) {
				t.Fatalf("customerfw must not define SCRM or industry domain model term %q in %s", term, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
