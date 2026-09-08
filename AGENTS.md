# AGENTS.md

Instructions for AI coding agents (Cursor, Claude Code, Codex, Copilot, Gemini, and similar). Humans: start at [README.md](README.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

Burrow is an independent copy of SaaS data on storage you control, that you can restore. The first loop is Gmail. Nothing is installable yet. Drive, Microsoft 365, Slack, GitHub, and Notion are roadmap names, not work.

## Hard rules

**TDD (red-green always).** Every production function under `packages/` starts as a failing test — `go test` for the engine and binaries, `bun test` for `packages/web`. Cycle: write the failing test → run it and see it fail for the right reason → write the minimum code → run it and see it pass. Bugs: reproduce with a failing test before the fix. No test required for docs, license, community files, `.gitkeep`, or this file.

**Test logic, not scaffolding.** The TDD rule above applies to behaviour that can be wrong in a way review would miss: invariants, orchestration, error paths, crash and restart safety, anything touching identity or the archive. It does not apply to static configuration, plain getters, struct wiring, input validators, or a library doing its documented job. If a test would only fail when the code is obviously broken, it is bloat — delete it.

**YAGNI.** Smallest change that satisfies the request. No extra connectors, packages, flags, or abstractions. No drive-by refactors. No microservices. Do not build a source because it appears in the README.

**Interface, then implementation.** A service (`internal/service/`) or a repository/adapter implementing a domain port exports an interface and returns it from its constructor — `NewFooService(...) FooService`, never the concrete type. The struct behind it is unexported (`fooService`, `objectRepository`) and callers depend on the interface only. `repository/ent/` and `internal/service/` already follow this; match it for anything new in either. Exception: `sqlite.Client` — the Database layer's single concrete owner of the handle (see the layer table below); `repository/factory.go`, not `Client`, is the seam for a future backend swap, so `Client` stays a concrete type.

**Comments earn their place.** The next reader is a developer. Do not restate what the code already says — no `// Fields of the X`, no `// NewFoo returns a Foo`, no narrating a loop. Write a comment only when it carries what the code cannot: why a non-obvious choice was made, a constraint the compiler will not enforce, or a trap the next change could spring. If deleting a comment loses nothing, delete it.

**Restore is the acceptance test.** A finished copy job is not done. If the change touches copy, verify, restore, or export, the test is that you got the object back.

**Verify before claiming done.** Run `make doctor`, `make test`, and `make lint` when they apply. If you did not run it, it does not work. Stubs that exit 1 are not a skip — say so.

**Do not invent.** Cite a file, a command, or say you do not know. Do not claim a connector, API, or make target exists.

**Issue before large design.** Open or comment on an issue before a new feature or architecture change. Small fixes can be a PR. See [CONTRIBUTING.md](CONTRIBUTING.md).

**Secrets stay out.** No tokens, keys, or customer data in the tree, logs, tests, or export dumps. Vulnerabilities: [SECURITY.md](SECURITY.md), never a public issue.

**Commits.** Do not commit unless the human asks. Then Conventional Commits, **one line only** — no body, no footer, no `Co-authored-by`, no AI trailer.

```text
type(scope): imperative summary
```

Types: `feat` `fix` `docs` `test` `refactor` `chore`. Scope is optional (`gmail`, `core`, `web`). Imperative, lowercase, no trailing period, ≤72 characters. Example: `docs: add AGENTS.md for coding agents`.

## Commands

Toolchain is [mise](https://mise.jdx.dev). `make` is the command surface. Bun is the runtime. Do not use npm or a global Node.

```bash
make install            # mise install + bun install
make doctor
make test               # go test ./... , then bun test when packages/web exists
make lint               # go vet, gofmt, layer check
make check              # lint then test
make migrate            # apply pending schema migrations
make migrate-dry-run    # print pending statements without applying
make generate-ent       # regenerate ent code from packages/engine/ent/schema
make clean              # remove build artifacts and dry-run dumps
make clean-archive      # DESTRUCTIVE: delete the local archive (needs CONFIRM=yes)
```

Verifying against a real archive writes to `~/.burrow`, which is the user's
data, not scratch space. Use a throwaway home instead: `HOME=$(mktemp -d) make migrate`.

## Layers (never violate)

| Layer | Path | Rule |
|---|---|---|
| Facade | `packages/engine/*.go` | The entire public surface. No `internal/` type in any signature. |
| Domain | `packages/engine/internal/domain/` | Models and interfaces. Zero third-party imports. No network, no disk. |
| Database | `packages/engine/internal/sqlite/` | Owns the handle, `WithTx`, and migrations. Nothing else opens a connection. |
| Repository | `packages/engine/internal/repository/` | `factory.go` is the wiring seam: callers depend on it, never on `repository/ent`. Implementations live in `repository/ent/`, where generated `ent.*` types never escape and queries go through `Querier(ctx)` so every method works inside a transaction. |
| Adapters | `internal/source/`, `internal/storage/`, `internal/credential/` | Implement ports. Never call upward. |
| Service | `packages/engine/internal/service/` | Use cases. Orchestrates repositories and adapters. Every service takes `service.ServiceParams` and nothing else, so a new dependency changes one struct rather than every constructor. |
| Binaries | `packages/engine/cmd/burrow/`, `packages/engine/cmd/migrate/` | Consume the facade only. The compiler cannot block `internal/` here — `scripts/check-layers.sh` does. |

- Migrations are versioned Atlas files. Never `Schema.Create` auto-migrate.
- Write order is mandatory: fsync the blob, then commit the row.
- `scripts/check-layers.sh` fails `make lint` if `domain/` imports anything third-party, or if `cmd/` imports `internal/`.

## Pointers

- Product and architecture: [README.md](README.md)
- Engine design, and why each decision was made: [docs/design/engine.md](docs/design/engine.md)
- What the IMAP spike measured: [docs/design/2026-09-08-imap-findings.md](docs/design/2026-09-08-imap-findings.md)
- License: [LICENSE](LICENSE) (AGPLv3 or later). Marks: [TRADEMARKS.md](TRADEMARKS.md)
- Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
