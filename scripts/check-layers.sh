#!/usr/bin/env bash
# Enforce the layer rules from AGENTS.md.
set -euo pipefail

fail=0

# 1. domain/ holds models and interfaces; it must import nothing third-party.
# grep exits 1 on no match, which is the success case here; go list failing is
# not. Capture them separately so a broken build cannot read as "no violations".
deps=$(go list -deps ./packages/engine/internal/domain/...)
domain_violations=$(printf '%s\n' "$deps" \
  | { grep -E '^[^/]+\.[^/]+/' || true; } \
  | { grep -v '^github.com/omkar273/burrow/' || true; })

if [ -n "$domain_violations" ]; then
  echo "internal/domain/ imports third-party packages:"
  echo "$domain_violations" | sed 's/^/  /'
  echo "domain holds models and interfaces only — move this to an adapter."
  fail=1
fi

# 2. cmd/ consumes the facade only.
#
# The compiler cannot enforce this: cmd/ sits under packages/engine/, so Go's
# internal rule permits it to reach internal/. The facade is the intended API
# and the reason the engine is embeddable, so the rule is enforced here instead.
cmd_violations=$(go list -f '{{.ImportPath}}{{range .Imports}} {{.}}{{end}}' \
  ./packages/engine/cmd/... \
  | tr ' ' '\n' \
  | { grep 'burrow/packages/engine/internal/' || true; } \
  | sort -u)

if [ -n "$cmd_violations" ]; then
  echo "packages/engine/cmd/ imports internal packages directly:"
  echo "$cmd_violations" | sed 's/^/  /'
  echo "binaries consume the facade in packages/engine/*.go — add to it instead."
  fail=1
fi

exit $fail
