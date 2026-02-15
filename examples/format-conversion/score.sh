#!/usr/bin/env bash
set -euo pipefail

# Score format conversion results.
# Usage: ./score.sh

cd "$(dirname "$0")"
go run . -score
