#!/usr/bin/env bash

# NexusCore Git Pre-Commit Hook
# Validates code syntax, styling, formats, linting, and runs unit tests before any commit is processed.
# Works across Linux, macOS, and Windows (WSL2 / Git Bash).

set -euo pipefail

# Visual markers
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

log_info() {
    echo -e "${YELLOW}[PRE-COMMIT]${NC} $1"
}

log_error() {
    echo -e "${RED}[PRE-COMMIT-ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PRE-COMMIT-SUCCESS]${NC} $1"
}

log_info "Executing NexusCore Workspace Pre-Commit Gates..."

# 1. Format verification
log_info "Verifying backend formatting (go fmt)..."
unformatted_files=$(gofmt -l .)
if [ -n "$unformatted_files" ]; then
    log_error "The following files contain formatting errors:"
    echo "$unformatted_files"
    log_error "Please execute 'make fmt' to format files before committing."
    exit 1
fi
log_success "Formatting verified successfully."

# 2. Workspace Linting
log_info "Executing golangci-lint checks..."
if command -v golangci-lint &> /dev/null; then
    if ! golangci-lint run --timeout=5m; then
        log_error "golangci-lint failed. Please fix violations."
        exit 1
    fi
else
    log_info "golangci-lint not installed locally, running fallback vetting (go vet)..."
    # Fallback to standard go vet on workspace modules
    go work use . || true
    if ! go vet ./...; then
        log_error "go vet verification failed."
        exit 1
    fi
fi
log_success "Lint checks passed successfully."

# 3. Unit Test suite execution
log_info "Executing unit test coverage suite..."
if ! go test -v -short -race ./...; then
    log_error "Unit tests failed. Commits must carry passing tests."
    exit 1
fi
log_success "All unit tests completed cleanly!"

log_success "All pre-commit gates satisfied. Code is approved for commit!"
exit 0
