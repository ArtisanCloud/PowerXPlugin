package templates

import "testing"

func TestNormalizeTargetPath_SelectsOnlyChosenBackendAndFrontend(t *testing.T) {
	tests := []struct {
		name         string
		rel          string
		backendType  string
		frontendType string
		wantPath     string
		wantKeep     bool
	}{
		{
			name:         "keep selected backend path",
			rel:          "backend/go-gin/internal/router/router.go",
			backendType:  "go-gin",
			frontendType: "nuxt",
			wantPath:     "backend/internal/router/router.go",
			wantKeep:     true,
		},
		{
			name:         "drop non selected backend path",
			rel:          "backend/python-fastapi/app/main.py",
			backendType:  "go-gin",
			frontendType: "nuxt",
			wantPath:     "",
			wantKeep:     false,
		},
		{
			name:         "keep selected frontend path",
			rel:          "web-admin/nuxt/app/pages/index.vue",
			backendType:  "go-gin",
			frontendType: "nuxt",
			wantPath:     "web-admin/app/pages/index.vue",
			wantKeep:     true,
		},
		{
			name:         "drop non selected frontend path",
			rel:          "web-admin/next/app/page.tsx",
			backendType:  "go-gin",
			frontendType: "nuxt",
			wantPath:     "",
			wantKeep:     false,
		},
		{
			name:         "keep non variant path",
			rel:          "README.md",
			backendType:  "go-gin",
			frontendType: "nuxt",
			wantPath:     "README.md",
			wantKeep:     true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			gotPath, gotKeep := normalizeTargetPath(testCase.rel, testCase.backendType, testCase.frontendType)
			if gotPath != testCase.wantPath {
				t.Fatalf("normalizeTargetPath() path = %q, want %q", gotPath, testCase.wantPath)
			}
			if gotKeep != testCase.wantKeep {
				t.Fatalf("normalizeTargetPath() keep = %v, want %v", gotKeep, testCase.wantKeep)
			}
		})
	}
}
