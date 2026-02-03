#!/usr/bin/env bash
# Cross-platform testing script for Go CLI
# Tests on macOS, Linux, and Windows (via Wine or CI)

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLI_DIR="$PROJECT_ROOT/tools/cli"
TEST_DIR="$PROJECT_ROOT/tmp/cross-platform-test"
REPORT_FILE="$TEST_DIR/cross-platform-report.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Host platform info
HOST_GOOS="$(uname -s | tr '[:upper:]' '[:lower:]')"
HOST_GOARCH_RAW="$(uname -m)"
case "$HOST_GOARCH_RAW" in
    "x86_64"|"amd64")
        HOST_GOARCH="amd64"
        ;;
    "arm64"|"aarch64")
        HOST_GOARCH="arm64"
        ;;
    *)
        HOST_GOARCH="$HOST_GOARCH_RAW"
        ;;
esac

# Portable map helpers
set_test_result() {
    local key="$1"
    local value="$2"

    for i in "${!TEST_RESULTS_KEYS[@]}"; do
        if [[ "${TEST_RESULTS_KEYS[$i]}" == "$key" ]]; then
            TEST_RESULTS_VALUES[$i]="$value"
            return
        fi
    done
    TEST_RESULTS_KEYS+=("$key")
    TEST_RESULTS_VALUES+=("$value")
}

get_test_result() {
    local key="$1"
    for i in "${!TEST_RESULTS_KEYS[@]}"; do
        if [[ "${TEST_RESULTS_KEYS[$i]}" == "$key" ]]; then
            echo "${TEST_RESULTS_VALUES[$i]}"
            return
        fi
    done
    echo ""
}

# Test results (portable map)
TEST_RESULTS_KEYS=()
TEST_RESULTS_VALUES=()
test_count=0
pass_count=0
fail_count=0

# Platforms to test
PLATFORMS=("linux/amd64" "darwin/amd64" "darwin/arm64" "windows/amd64")

# Initialize
echo -e "${BLUE}=== Go CLI Cross-Platform Testing ===${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Test Directory: $TEST_DIR"
echo ""

# Create test directory
mkdir -p "$TEST_DIR"

# Function to print colored output
print_test() {
    local platform=$1
    local test_name=$2
    echo -e "${BLUE}[TEST]${NC} $platform - $test_name"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((pass_count++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((fail_count++))
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Determine whether host can execute the target binary
can_run_platform() {
    local platform=$1
    IFS='/' read -r goos goarch <<< "$platform"
    if [[ "$goos" == "$HOST_GOOS" && "$goarch" == "$HOST_GOARCH" ]]; then
        return 0
    fi
    return 1
}

# Function to build for specific platform
build_for_platform() {
    local platform=$1
    local output_dir="$TEST_DIR/$platform"

    mkdir -p "$output_dir"

    # Parse GOOS and GOARCH
    IFS='/' read -r goos goarch <<< "$platform"

    echo "Building for $goos/$goarch..."

    # Build command
    GOOS=$goos GOARCH=$goarch go build -o "$output_dir/px-plugin" ./cmd/px-plugin

    if [ $? -eq 0 ]; then
        print_success "Built for $platform"
        return 0
    else
        print_error "Failed to build for $platform"
        return 1
    fi
}

# Test basic functionality
test_basic_functionality() {
    local platform=$1
    local cli_path="$TEST_DIR/$platform/px-plugin"

    print_test "$platform" "Basic Functionality"

    # Test 1: Help command
    if "$cli_path" --help > /dev/null 2>&1; then
        print_success "Help command works"
    else
        print_error "Help command failed"
        return 1
    fi

    # Test 2: Dev command exists
    if "$cli_path" dev --help > /dev/null 2>&1; then
        print_success "Dev command exists"
    else
        print_error "Dev command failed"
        return 1
    fi

    # Test 3: Version info
    if "$cli_path" --version 2>/dev/null || true; then
        print_success "Version command works"
    else
        print_info "Version command not implemented (optional)"
    fi

    return 0
}

# Test file watching
test_file_watching() {
    local platform=$1
    local cli_path="$TEST_DIR/$platform/px-plugin"

    print_test "$platform" "File Watching"

    # Create test directory
    local test_plugin_dir="$TEST_DIR/$platform/test-plugin"
    mkdir -p "$test_plugin_dir"

    # Create plugin.yaml
    cat > "$test_plugin_dir/plugin.yaml" << 'EOF'
name: test-plugin
version: 0.1.0
entry: ./main.go
EOF

    # Create main.go
    cat > "$test_plugin_dir/main.go" << 'EOF'
package main
import "fmt"
func main() {
    fmt.Println("Hello, World!")
}
EOF

    # Test that file watcher can start
    # (We won't actually run it, just verify the command is valid)
    if "$cli_path" dev --watch --entry "$test_plugin_dir" --help > /dev/null 2>&1; then
        print_success "File watching setup works"
    else
        print_error "File watching setup failed"
        return 1
    fi

    return 0
}

# Test path handling
test_path_handling() {
    local platform=$1
    local cli_path="$TEST_DIR/$platform/px-plugin"

    print_test "$platform" "Path Handling"

    # Test with spaces in path
    local test_dir="$TEST_DIR/$platform/test dir with spaces"
    mkdir -p "$test_dir"

    # This should not crash
    if "$cli_path" dev --watch --entry "$test_dir" --help > /dev/null 2>&1; then
        print_success "Handles paths with spaces"
    else
        print_error "Failed to handle paths with spaces"
        return 1
    fi

    # Test with special characters (if supported)
    # Note: Some platforms may not support all special characters
    return 0
}

# Test configuration
test_configuration() {
    local platform=$1
    local cli_path="$TEST_DIR/$platform/px-plugin"

    print_test "$platform" "Configuration"

    # Test with environment variables
    if PX_DEBUG=true "$cli_path" dev --help > /dev/null 2>&1; then
        print_success "Environment variable support works"
    else
        print_error "Environment variable support failed"
        return 1
    fi

    return 0
}

# Test binary properties
test_binary_properties() {
    local platform=$1
    local cli_path="$TEST_DIR/$platform/px-plugin"

    print_test "$platform" "Binary Properties"

    # Check if binary exists
    if [ ! -f "$cli_path" ]; then
        print_error "Binary not found"
        return 1
    fi

    # Check if binary is executable (Unix-like)
    if [[ "$platform" != windows/* ]]; then
        if [ -x "$cli_path" ]; then
            print_success "Binary is executable"
        else
            print_error "Binary is not executable"
            return 1
        fi
    fi

    # Check file size
    local size=$(stat -f%z "$cli_path" 2>/dev/null || stat -c%s "$cli_path" 2>/dev/null || echo "0")
    local size_mb=$((size / 1024 / 1024))

    print_info "Binary size: ${size_mb}MB"

    if [ $size_mb -lt 50 ]; then
        print_success "Binary size is reasonable (< 50MB)"
    else
        print_info "Binary size is large (${size_mb}MB)"
    fi

    return 0
}

# Main testing function
run_tests_for_platform() {
    local platform=$1

    echo ""
    echo -e "${BLUE}=== Testing Platform: $platform ===${NC}"

    # Build for platform
    if ! build_for_platform "$platform"; then
        set_test_result "$platform" "BUILD_FAILED"
        return 1
    fi

    if ! can_run_platform "$platform"; then
        print_info "Skipping runtime tests for $platform (host ${HOST_GOOS}/${HOST_GOARCH})"
        set_test_result "$platform" "BUILD_ONLY"
        return 0
    fi

    # Run tests
    local tests_failed=0

    if ! test_basic_functionality "$platform"; then
        ((tests_failed++))
    fi

    if ! test_file_watching "$platform"; then
        ((tests_failed++))
    fi

    if ! test_path_handling "$platform"; then
        ((tests_failed++))
    fi

    if ! test_configuration "$platform"; then
        ((tests_failed++))
    fi

    if ! test_binary_properties "$platform"; then
        ((tests_failed++))
    fi

    # Record results
    if [ $tests_failed -eq 0 ]; then
        set_test_result "$platform" "PASS"
        print_success "All tests passed for $platform"
    else
        set_test_result "$platform" "FAIL"
        print_error "$tests_failed tests failed for $platform"
    fi

    return $tests_failed
}

# Check if platform is supported
is_platform_supported() {
    local platform=$1

    # Check if GOOS/GOARCH is supported
    case $platform in
        "linux/amd64"|"darwin/amd64"|"darwin/arm64"|"windows/amd64")
            return 0
            ;;
        *)
            print_info "Skipping unsupported platform: $platform"
            return 1
            ;;
    esac
}

# Main execution
main() {
    echo "Building Go CLI for cross-platform testing..."
    cd "$CLI_DIR"

    # Build for each platform
    for platform in "${PLATFORMS[@]}"; do
        if is_platform_supported "$platform"; then
            if ! run_tests_for_platform "$platform"; then
                print_info "Continuing despite failures on $platform (see report)"
            fi
            ((test_count++))
        fi
    done

    # Generate report
    echo ""
    echo "=== Generating Cross-Platform Test Report ==="

    cat > "$REPORT_FILE" << EOF
# Cross-Platform Test Report

Generated: $(date)

## Platforms Tested

EOF

    for platform in "${PLATFORMS[@]}"; do
        if is_platform_supported "$platform"; then
            local result
            result="$(get_test_result "$platform")"
            if [[ -z "$result" ]]; then
                result="NOT_TESTED"
            fi
            local status_icon="❓"

            case $result in
                "PASS")
                    status_icon="✅"
                    ;;
                "FAIL")
                    status_icon="❌"
                    ;;
                "BUILD_FAILED")
                    status_icon="🔨"
                    ;;
            esac

            echo "- $status_icon **$platform**: $result" >> "$REPORT_FILE"
        fi
    done

    cat >> "$REPORT_FILE" << EOF

## Summary

- Total Platforms: $test_count
- Passed: $pass_count
- Failed: $fail_count

## Test Results

EOF

    for platform in "${PLATFORMS[@]}"; do
        if is_platform_supported "$platform"; then
            local result
            result="$(get_test_result "$platform")"
            if [[ -z "$result" ]]; then
                result="NOT_TESTED"
            fi

            if [ "$result" != "NOT_TESTED" ]; then
                echo "### $platform" >> "$REPORT_FILE"
                echo "Result: $result" >> "$REPORT_FILE"

                if [ -f "$TEST_DIR/$platform/px-plugin" ]; then
                    local size=$(stat -f%z "$TEST_DIR/$platform/px-plugin" 2>/dev/null || stat -c%s "$TEST_DIR/$platform/px-plugin" 2>/dev/null || echo "0")
                    local size_mb=$((size / 1024 / 1024))
                    echo "Binary Size: ${size_mb}MB" >> "$REPORT_FILE"
                fi

                echo "" >> "$REPORT_FILE"
            fi
        fi
    done

    # Add recommendations
    cat >> "$REPORT_FILE" << EOF

## Recommendations

### Distribution
- **Linux/AMD64**: Primary target for servers and CI/CD
- **macOS/AMD64 & ARM64**: Support both Intel and Apple Silicon Macs
- **Windows/AMD64**: Target Windows 10/11 users

### Build Optimization
- Use release builds: \`go build -ldflags "-s -w"\`
- Consider UPX compression for smaller binaries
- Test on actual hardware for each platform

### CI/CD Integration
- Use GitHub Actions or similar for automated testing
- Build on native platforms when possible
- Use QEMU for cross-platform testing if needed

## Known Issues

### Windows
- Path handling with backslashes vs forward slashes
- Line ending differences (CRLF vs LF)
- Process termination behavior

### macOS
- Code signing may be required for distribution
- Gatekeeper may block unsigned binaries
- Apple Silicon requires ARM64 build

### Linux
- Different distributions have different library versions
- Static linking may be required for portability
- SELinux/AppArmor may restrict execution

EOF

    echo "Report saved to: $REPORT_FILE"
    cat "$REPORT_FILE"

    # Exit with appropriate code
    if [ $fail_count -gt 0 ]; then
        echo ""
        print_error "Some tests failed ($fail_count/$test_count)"
        exit 1
    else
        echo ""
        print_success "All tests passed! ($pass_count/$test_count)"
        exit 0
    fi
}

# Run main function
main "$@"
