set shell := ["bash", "-uc"]
VERSION := "dev"
GO_LDFLAGS := "-s -w -X main.version={{ VERSION }}"
GO_FLAGS := "-trimpath"

default: check

lint:
    golangci-lint run

test-unit:
    go test ./...

test-race:
    go test -race ./...

test-integration:
    go run ./scripts/mcp_smoke.go

test: test-unit test-race test-integration

check: lint test

build-linux:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build {{ GO_FLAGS }} -ldflags={{ GO_LDFLAGS }} -o ./bin/linux/mnemonic ./cmd/mnemonic

build-macos:
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build {{ GO_FLAGS }} -ldflags={{ GO_LDFLAGS }} -o ./bin/macos/mnemonic ./cmd/mnemonic

build: clean build-linux build-macos

archive-linux: build-linux
    tar -czf ./bin/linux/mnemonic_{{ VERSION }}_linux_amd64.tar.gz -C ./bin/linux mnemonic README.md LICENSE* 2>/dev/null || tar -czf ./bin/linux/mnemonic_{{ VERSION }}_linux_amd64.tar.gz -C ./bin/linux mnemonic

archive-macos: build-macos
    tar -czf ./bin/macos/mnemonic_{{ VERSION }}_darwin_arm64.tar.gz -C ./bin/macos mnemonic README.md LICENSE* 2>/dev/null || tar -czf ./bin/macos/mnemonic_{{ VERSION }}_darwin_arm64.tar.gz -C ./bin/macos mnemonic

archive: archive-linux archive-macos

build-linux-debug:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags="all=-N -l" -o ./bin/linux/mnemonic_linux_amd64_debug ./cmd/mnemonic

build-macos-debug:
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -gcflags="all=-N -l" -o ./bin/macos/mnemonic_darwin_arm64_debug ./cmd/mnemonic

build-debug: build-linux-debug build-macos-debug

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

collect-content-mds:
    collect_content . --skip-empty --format md --sort dirs-first --ext ".md" --exclude "testdata" > .tmp/mnemonic_mds.md

collect-open-issues:
    gh issue list --state open --json number,title,body | jq -r '.[] | "#\(.number) \(.title)\n\(.body | split("\n") | join("\n"))\n"' > .tmp/open_issues.md
