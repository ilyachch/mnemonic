set shell := ["bash", "-uc"]

default: check

fmt:
    go fmt ./...

vet:
    go vet ./...

lint: fmt vet

test-unit:
    go test ./...

test-race:
    go test -race ./...

test-integration:
    go run ./scripts/mcp_smoke.go

test: test-unit test-race test-integration

check: lint test

build-linux:
    GOOS=linux GOARCH=amd64 go build -o ./bin/linux/mnemonic ./cmd/mnemonic

build-macos:
    GOOS=darwin GOARCH=arm64 go build -o ./bin/macos/mnemonic ./cmd/mnemonic

build: clean build-linux build-macos

install:
    go install ./cmd/mnemonic

clean:
    rm -rf ./bin ./.tmp
    rm -f ./coverage.out ./coverage.html

[private]
_run-coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "=== Running tests with coverage ==="
    go test -coverprofile=coverage.out -covermode=atomic ./...

    if [ -f coverage.out ]; then
        grep -vE "internal/testutil|scripts/mcp_smoke\.go" coverage.out > coverage.filtered.out
        mv coverage.filtered.out coverage.out
    fi

coverage: _run-coverage
    go tool cover -html=coverage.out -o coverage.html

coverage-report: _run-coverage
    go tool cover -func=coverage.out

coverage-check threshold="70.0": _run-coverage
    #!/usr/bin/env bash
    set -euo pipefail

    MIN_COVERAGE="{{ threshold }}"
    echo "=== Checking coverage threshold ==="

    TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

    echo "Total coverage: ${TOTAL_COVERAGE}%"
    echo "Minimum threshold: ${MIN_COVERAGE}%"

    if awk "BEGIN { exit !(${TOTAL_COVERAGE} < ${MIN_COVERAGE}) }"; then
        echo ""
        echo "ERROR: Code coverage (${TOTAL_COVERAGE}%) is below the minimum threshold (${MIN_COVERAGE}%)!"
        exit 1
    fi

    echo ""
    echo "SUCCESS: Code coverage (${TOTAL_COVERAGE}%) meets the minimum threshold (${MIN_COVERAGE}%)."

collect-content:
    collect_content . --skip-empty --format md --sort dirs-first --ext ".go" > .tmp/mnemonic.md

collect-content-no-tests:
    collect_content . --skip-empty --format md --sort dirs-first --ext ".go" --exclude "*_test.go" > .tmp/mnemonic_no_tests.md
