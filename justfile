set shell := ["bash", "-uc"]
VERSION := "dev"
LD_FLAGS_STR := "-s -w -X main.version=" + VERSION

default: check

[private]
_mkdir_tmp:
    mkdir -p ./tmp

[private]
_mkdir_dist:
    mkdir -p ./dist

[private]
_mkdir_bin:
    mkdir -p ./bin/linux ./bin/macos

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

build-cli-docs:
    go run ./cmd/docs > README.cli.md

build-linux: _mkdir_bin build-cli-docs
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="{{ LD_FLAGS_STR }}" -o ./bin/linux/mnemonic ./cmd/mnemonic

build-macos: _mkdir_bin build-cli-docs
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="{{ LD_FLAGS_STR }}" -o ./bin/macos/mnemonic ./cmd/mnemonic

build: clean build-linux build-macos

archive-linux: _mkdir_dist build-linux
	tar -czf ./dist/mnemonic_{{ VERSION }}_linux_amd64.tar.gz README* LICENSE* PROMPTS* skills -C ./bin/linux mnemonic

archive-macos: _mkdir_dist build-macos
	tar -czf ./dist/mnemonic_{{ VERSION }}_darwin_arm64.tar.gz README* LICENSE* PROMPTS* skills -C ./bin/macos mnemonic

archive: archive-linux archive-macos

prepare-packages: build archive

build-linux-debug: _mkdir_bin
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags="all=-N -l" -o ./bin/linux/mnemonic_linux_amd64_debug ./cmd/mnemonic

build-macos-debug: _mkdir_bin
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -gcflags="all=-N -l" -o ./bin/macos/mnemonic_darwin_arm64_debug ./cmd/mnemonic

build-debug: build-linux-debug build-macos-debug

install:
    go install ./cmd/mnemonic

clean:
    rm -rf ./bin ./tmp ./dist
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


collect-content: _mkdir_tmp
    collect_content . --skip-empty --format md --sort dirs-first --ext ".go" > ./tmp/mnemonic.md

collect-content-no-tests: _mkdir_tmp
    collect_content . --skip-empty --format md --sort dirs-first --ext ".go" --exclude "*_test.go" > ./tmp/mnemonic_no_tests.md

collect-content-mds: _mkdir_tmp
    collect_content . --skip-empty --format md --sort dirs-first --ext ".md" --exclude "testdata" "tmp" > ./tmp/mnemonic_mds.md

collect-open-issues: _mkdir_tmp
    gh issue list --state open --json number,title,body | jq -r '.[] | "#\(.number) \(.title)\n\(.body | split("\n") | join("\n"))\n"' > ./tmp/open_issues.md
