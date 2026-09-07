#!/usr/bin/env bash
# Enforce the layer rule from AGENTS.md: internal/domain/ must import
# nothing third-party.
set -euo pipefail

violations=$(go list -deps ./packages/engine/internal/domain/... \
  | grep -E '^[^/]+\.[^/]+/' \
  | grep -v '^github.com/omkar273/burrow/' \
  || true)

if [ -n "$violations" ]; then
  echo "internal/domain/ imports third-party packages:"
  echo "$violations" | sed 's/^/  /'
  echo "domain holds models and interfaces only — move this to an adapter."
  exit 1
fi
