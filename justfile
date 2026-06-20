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
    rm -rf ./bin

coverage:
    go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html

coverage-report:
    go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

collect-content:
    collect_content . --skip-empty --format md --sort dirs-first --ext ".go" > mnemonic.md

# Сборка содержимого исходного кода (без тестов) в один Markdown-файл
collect-content-no-tests:
    collect_content . --skip-empty --format md --sort dirs-first --ext ".go" --exclude "*_test.go" > mnemonic_no_tests.md
