package templates

// Template types constants for backend and frontend frameworks.
// These constants should be used throughout the codebase to avoid magic strings
// and make it easier to add new template types in the future.
const (
	// Backend template types
	BackendGoGin      = "go-gin"
	BackendGoFiber    = "go-fiber" // Future support
	BackendGoEcho     = "go-echo"  // Future support
	BackendGoChi      = "go-chi"   // Future support

	// Frontend template types
	FrontendNuxt      = "nuxt"     // Current support
	FrontendNext      = "next"     // Future support
	FrontendVue       = "vue"      // Future support
	FrontendReact     = "react"    // Future support
	FrontendSvelte    = "svelte"   // Future support
)

// SupportedBackends returns a list of all supported backend template types.
func SupportedBackends() []string {
	return []string{
		BackendGoGin,
		// BackendGoFiber,
		// BackendGoEcho,
		// BackendGoChi,
	}
}

// SupportedFrontends returns a list of all supported frontend template types.
func SupportedFrontends() []string {
	return []string{
		FrontendNuxt,
		// FrontendNext,
		// FrontendVue,
		// FrontendReact,
		// FrontendSvelte,
	}
}

// IsValidBackend checks if the given backend type is supported.
func IsValidBackend(backend string) bool {
	for _, supported := range SupportedBackends() {
		if supported == backend {
			return true
		}
	}
	return false
}

// IsValidFrontend checks if the given frontend type is supported.
func IsValidFrontend(frontend string) bool {
	for _, supported := range SupportedFrontends() {
		if supported == frontend {
			return true
		}
	}
	return false
}
