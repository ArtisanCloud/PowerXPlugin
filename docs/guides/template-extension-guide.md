# Template Extension Guide

This guide explains how to add new template types to the px-plugin CLI.

## Overview

The template system uses constants defined in `tools/cli/internal/templates/constants.go` to manage supported backend and frontend frameworks. This centralized approach makes it easy to add new templates without modifying multiple files.

## Command-Line Interface

The new CLI design uses clear, descriptive flags to specify frameworks:

```bash
px-plugin init \
  --backend go-gin \      # Backend framework
  --admin nuxt \          # Admin frontend framework
  --app vue \             # Optional app frontend framework
  com.example.myplugin
```

**Parameters:**

- `--backend`: Backend framework (required, default: go-gin)
- `--admin`: Admin frontend framework (required, default: nuxt)
- `--app`: App frontend framework (optional, defaults to --admin value)

## Adding a New Backend Template

### Step 1: Add Constants

Edit `tools/cli/internal/templates/constants.go`:

```go
// Add new backend constant
BackendGoFiber = "go-fiber"  // Add this line
BackendGoEcho  = "go-echo"   // Add this line
BackendGoChi   = "go-chi"    // Add this line

// Uncomment in SupportedBackends()
func SupportedBackends() []string {
	return []string{
		BackendGoGin,
		BackendGoFiber,    // Uncomment
		// BackendGoEcho,  // Uncomment when ready
		// BackendGoChi,   // Uncomment when ready
	}
}
```

### Step 2: Create Template Directory

Create the template directory structure:

```bash
mkdir -p tools/cli/internal/templates/data/backend/go-fiber
# Copy files from go-gin and modify as needed
cp -r tools/cli/internal/templates/data/backend/go-gin/* tools/cli/internal/templates/data/backend/go-fiber/
```

### Step 3: Update Template Files

Modify the template files in `backend/go-fiber/` to work with the new framework:

- Update `go.mod` dependencies
- Modify main.go and other source files
- Update configuration files
- Adjust package names and imports

### Step 4: Test

```bash
go build -o ./tmp/px-plugin ./tools/cli/cmd/px-plugin/main.go
./tmp/px-plugin init --backend go-fiber com.example.newplugin
```

## Adding a New Frontend Template

### Step 1: Add Constants

Edit `tools/cli/internal/templates/constants.go`:

```go
// Add new frontend constant
FrontendNext      = "next"     // Add this line
FrontendReact     = "react"    // Add this line
FrontendVue       = "vue"      // Add this line
FrontendSvelte    = "svelte"   // Add this line

// Uncomment in SupportedFrontends()
func SupportedFrontends() []string {
	return []string{
		FrontendNuxt,
		FrontendNext,     // Uncomment
		// FrontendReact,  // Uncomment when ready
		// FrontendVue,    // Uncomment when ready
		// FrontendSvelte, // Uncomment when ready
	}
}
```

### Step 2: Create Template Directory

```bash
mkdir -p tools/cli/internal/templates/data/web-admin/next
# Copy files from nuxt and modify as needed
cp -r tools/cli/internal/templates/data/web-admin/nuxt/* tools/cli/internal/templates/data/web-admin/next/
```

### Step 3: Update Template Files

Modify the template files in `web-admin/next/`:

- Update `package.json` dependencies
- Modify Vue/React components
- Update build configuration
- Adjust routing and structure

### Step 4: Test

```bash
go build -o ./tmp/px-plugin ./tools/cli/cmd/px-plugin/main.go
./tmp/px-plugin init --admin next com.example.newplugin
```

## Multiple Frontend Support

The CLI now supports two frontend frameworks:

- `--admin`: For the admin/management interface
- `--app`: For the end-user application (optional)

This allows creating plugins with different frontend frameworks:

```bash
# Admin and app use different frameworks
px-plugin init --backend go-gin --admin nuxt --app vue com.example.myplugin

# Both use the same framework (default behavior)
px-plugin init --backend go-gin --admin nuxt com.example.myplugin
# The --app flag defaults to the same value as --admin
```

## Best Practices

1. **Follow Naming Conventions**

   - Backend: `go-<framework-name>` (e.g., `go-gin`, `go-fiber`)
   - Frontend: `<framework-name>` (e.g., `nuxt`, `next`, `react`)

2. **Template Structure**

   - Keep templates in `tools/cli/internal/templates/data/`
   - Backend templates in `backend/<type>/`
   - Frontend templates in `web-admin/<type>/`

3. **Testing**

   - Always test with `--force` flag
   - Verify generated code compiles
   - Check all files are created correctly

4. **Documentation**

   - Update this guide
   - Update CLI help output (root.go)
   - Add examples in README

## Example: Adding go-fiber Backend

Here's a complete example of adding `go-fiber` support:

### constants.go changes:

```go
const (
	BackendGoGin   = "go-gin"
	BackendGoFiber = "go-fiber"  // NEW
	BackendGoEcho  = "go-echo"   // Future
	BackendGoChi   = "go-chi"    // Future

	FrontendNuxt  = "nuxt"   // Current
	FrontendNext  = "next"   // Future
	FrontendReact = "react"  // Future
)

func SupportedBackends() []string {
	return []string{
		BackendGoGin,
		BackendGoFiber,  // Enable
		// BackendGoEcho,
		// BackendGoChi,
	}
}
```

### Template files (backend/go-fiber/go.mod):

```go
module github.com/ArtisanCloud/PowerXPlugin/plugins/{{ .PluginID }}/backend

go {{ .GoVersion }}

require (
	github.com/gofiber/fiber/v2 v2.49.0
	github.com/gofiber/fiber/v2/middleware/cors v1.3.0
	// ... other dependencies
)
```

## Verification Checklist

After adding a new template, verify:

1. ✅ Constants are defined correctly
2. ✅ Supported lists include the new template
3. ✅ Template directory exists with files
4. ✅ Template files use correct variable syntax (`{{ .VariableName }}`)
5. ✅ `px-plugin init --backend <type>` works
6. ✅ `px-plugin init --admin <type>` works
7. ✅ Generated code compiles
8. ✅ Help output shows new template as available

## Troubleshooting

### Template not found

- Check directory path: `tools/cli/internal/templates/data/backend/<type>/`
- Verify directory name matches constant value exactly

### Variables not replaced

- Ensure template files end with `.tmpl`
- Use `{{ .VariableName }}` syntax
- Check variable is in `templates.Data` struct

### Build errors

- Verify `go.mod` has correct module path
- Check imports and dependencies
- Ensure version numbers are valid

### Runtime errors

- Check file permissions
- Verify all required files are copied
- Test with `--force` flag
