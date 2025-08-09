#\!/bin/bash

# Test suite for check_coverage.sh script
# Following TDD methodology - these tests define the expected behavior

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CHECK_COVERAGE_SCRIPT="$PROJECT_ROOT/scripts/check_coverage.sh"

# Test helper functions
setup_test() {
    export TEST_MODE=1
    export TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    # Create minimal Go module for testing
    cat > go.mod << 'GOMOD'
module testmodule
go 1.21
GOMOD
}

cleanup_test() {
    cd "$PROJECT_ROOT"
    rm -rf "$TEMP_DIR"
}

# Test 1: Script should exist and be executable
test_script_exists() {
    echo "Testing: Script exists and is executable"
    
    if [ \! -f "$CHECK_COVERAGE_SCRIPT" ]; then
        echo "FAIL: check_coverage.sh does not exist"
        return 1
    fi
    
    if [ \! -x "$CHECK_COVERAGE_SCRIPT" ]; then
        echo "FAIL: check_coverage.sh is not executable"
        return 1
    fi
    
    echo "PASS: Script exists and is executable"
    return 0
}

# Test 2: Script should run go test -cover command
test_runs_coverage_command() {
    echo "Testing: Script runs go test -cover command"
    
    setup_test
    
    # Create a simple test file with 80% coverage (above threshold)
    cat > main.go << 'MAIN'
package main
func Add(a, b int) int { return a + b }
func Subtract(a, b int) int { return a - b }  // Not covered
func main() {}
MAIN
    
    cat > main_test.go << 'TEST'
package main
import "testing"
func TestAdd(t *testing.T) {
    if Add(2, 3) \!= 5 {
        t.Error("Add function failed")
    }
}
TEST
    
    # Run the script and check it executes go test -cover
    output=$("$CHECK_COVERAGE_SCRIPT" 2>&1 || true)
    
    cleanup_test
    
    # The output should contain coverage information
    if echo "$output" | grep -q "coverage:"; then
        echo "PASS: Script runs coverage command"
        return 0
    else
        echo "FAIL: Script does not run coverage command"
        echo "Output: $output"
        return 1
    fi
}

# Test 3: Script should pass when coverage meets threshold
test_passes_with_high_coverage() {
    echo "Testing: Script passes when coverage >= 70%"
    
    setup_test
    
    # Create test with high coverage (should pass)
    cat > main.go << 'MAIN'
package main
func Add(a, b int) int { return a + b }
func main() {}
MAIN
    
    cat > main_test.go << 'TEST'
package main
import "testing"
func TestAdd(t *testing.T) {
    if Add(2, 3) \!= 5 {
        t.Error("Add function failed")
    }
}
TEST
    
    # Run the script - it should exit with code 0 for high coverage
    if "$CHECK_COVERAGE_SCRIPT" >/dev/null 2>&1; then
        echo "PASS: Script passes with high coverage"
        cleanup_test
        return 0
    else
        echo "FAIL: Script should pass with high coverage"
        cleanup_test
        return 1
    fi
}

# Test 4: Script should fail when coverage is below threshold
test_fails_with_low_coverage() {
    echo "Testing: Script fails when coverage < 70%"
    
    setup_test
    
    # Create test with low coverage (should fail)
    cat > main.go << 'MAIN'
package main
func Add(a, b int) int { return a + b }
func Subtract(a, b int) int { return a - b }
func Multiply(a, b int) int { return a * b }
func Divide(a, b int) int { return a / b }
func main() {}
MAIN
    
    cat > main_test.go << 'TEST'
package main
import "testing"
func TestAdd(t *testing.T) {
    if Add(2, 3) \!= 5 {
        t.Error("Add function failed")
    }
}
TEST
    
    # Run the script - it should exit with non-zero code for low coverage
    if \! "$CHECK_COVERAGE_SCRIPT" >/dev/null 2>&1; then
        echo "PASS: Script fails with low coverage"
        cleanup_test
        return 0
    else
        echo "FAIL: Script should fail with low coverage"
        cleanup_test
        return 1
    fi
}

# Test 5: Script should provide clear output about coverage status
test_provides_clear_output() {
    echo "Testing: Script provides clear coverage status output"
    
    setup_test
    
    # Create test with known coverage
    cat > main.go << 'MAIN'
package main
func Add(a, b int) int { return a + b }
func main() {}
MAIN
    
    cat > main_test.go << 'TEST'
package main
import "testing"
func TestAdd(t *testing.T) {
    if Add(2, 3) \!= 5 {
        t.Error("Add function failed")
    }
}
TEST
    
    output=$("$CHECK_COVERAGE_SCRIPT" 2>&1 || true)
    
    cleanup_test
    
    # Output should contain coverage percentage and threshold information
    if echo "$output" | grep -q "%"; then
        echo "PASS: Script provides coverage percentage output"
        return 0
    else
        echo "FAIL: Script should provide clear coverage output"
        echo "Output: $output"
        return 1
    fi
}

# Run all tests
main() {
    echo "Running check_coverage.sh test suite..."
    echo "========================================"
    
    local failures=0
    
    test_script_exists || ((failures++))
    test_runs_coverage_command || ((failures++))
    test_passes_with_high_coverage || ((failures++))
    test_fails_with_low_coverage || ((failures++))
    test_provides_clear_output || ((failures++))
    
    echo "========================================"
    if [ $failures -eq 0 ]; then
        echo "All tests passed\!"
        exit 0
    else
        echo "$failures test(s) failed\!"
        exit 1
    fi
}

# Run tests if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
EOF < /dev/null