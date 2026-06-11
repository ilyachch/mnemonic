#!/usr/bin/env bash
set -euo pipefail

# Default minimum coverage threshold (can be overridden via COVERAGE_THRESHOLD env var)
MIN_COVERAGE="${COVERAGE_THRESHOLD:-70.0}"

echo "=== Running tests with coverage ==="
go test -coverprofile=coverage.out -covermode=atomic ./...

echo ""
echo "=== Coverage report ==="
go tool cover -func=coverage.out

echo ""
TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "Total coverage: ${TOTAL_COVERAGE}%"
echo "Minimum threshold: ${MIN_COVERAGE}%"

# Use awk for floating-point comparison (no bc dependency)
if awk "BEGIN { exit !(${TOTAL_COVERAGE} < ${MIN_COVERAGE}) }"; then
    echo ""
    echo "ERROR: Code coverage (${TOTAL_COVERAGE}%) is below the minimum threshold (${MIN_COVERAGE}%)!"
    exit 1
fi

echo ""
echo "SUCCESS: Code coverage (${TOTAL_COVERAGE}%) meets the minimum threshold (${MIN_COVERAGE}%)."