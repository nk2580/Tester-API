#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

THRESHOLD="${COVERAGE_THRESHOLD:-95}"
COVER_PROFILE="${COVER_PROFILE:-.tmp/coverage.out}"

mkdir -p .tmp/gomodcache .tmp/gocache

GOMODCACHE="$ROOT_DIR/.tmp/gomodcache" \
GOCACHE="$ROOT_DIR/.tmp/gocache" \
go test ./... -coverprofile="$COVER_PROFILE"

COVER_REPORT="$(go tool cover -func="$COVER_PROFILE")"
TOTAL="$(printf '%s\n' "$COVER_REPORT" | awk '/^total:/ {print $3}' | tr -d '%')"

if [[ -z "$TOTAL" ]]; then
  echo "Failed to parse total coverage from profile $COVER_PROFILE" >&2
  exit 1
fi

awk -v total="$TOTAL" -v threshold="$THRESHOLD" 'BEGIN {
  if (total + 0 < threshold + 0) {
    printf("Coverage gate failed: %.1f%% < %.1f%%\n", total, threshold)
    exit 1
  }
  printf("Coverage gate passed: %.1f%% >= %.1f%%\n", total, threshold)
}'
