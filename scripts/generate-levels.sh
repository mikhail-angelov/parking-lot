#!/usr/bin/env bash
# Regenerates the level packs under public/levels/. Leaves changes uncommitted.
# Usage: scripts/generate-levels.sh [--packs N] [--pack-size N] [--seed N] [--workers N]
set -euo pipefail
cd "$(dirname "$0")/.."

go run ./cmd/parking generate-packs \
  --out public/levels \
  --packs "${PACKS:-6}" \
  --pack-size "${PACK_SIZE:-100}" \
  --seed "${SEED:-1}" \
  --workers "${WORKERS:-4}" \
  "$@"
