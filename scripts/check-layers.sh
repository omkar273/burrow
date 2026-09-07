#!/usr/bin/env bash
# Enforce the layer rule from AGENTS.md: internal/domain/ must import
# nothing third-party.
set -euo pipefail

# grep exits 1 on no match, which is the success case here; go list failing is
# not. Capture them separately so a broken build cannot read as "no violations".
deps=$(go list -deps ./packages/engine/internal/domain/...)
violations=$(printf '%s\n' "$deps" \
  | { grep -E '^[^/]+\.[^/]+/' || true; } \
  | { grep -v '^github.com/omkar273/burrow/' || true; })

if [ -n "$violations" ]; then
  echo "internal/domain/ imports third-party packages:"
  echo "$violations" | sed 's/^/  /'
  echo "domain holds models and interfaces only — move this to an adapter."
  exit 1
fi
