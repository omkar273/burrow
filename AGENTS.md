# AGENTS.md

Instructions for AI coding agents (Cursor, Claude Code, Codex, Copilot, Gemini, and similar). Humans: start at [README.md](README.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

Burrow is an independent copy of SaaS data on storage you control, that you can restore. The first loop is Gmail. Nothing is installable yet. Drive, Microsoft 365, Slack, GitHub, and Notion are roadmap names, not work.

## Hard rules

**TDD (red-green always).** Every production function under `apps/` and `packages/` starts as a failing `bun test`. Cycle: write the failing test → run it and see it fail for the right reason → write the minimum code → run it and see it pass. Bugs: reproduce with a failing test before the fix. No test required for docs, license, community files, `.gitkeep`, or this file.

**YAGNI.** Smallest change that satisfies the request. No extra connectors, packages, flags, or abstractions. No drive-by refactors. No microservices. Do not build a source because it appears in the README.

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
make install    # mise install + bun install
make doctor
make test       # bun test (stub until apps exist)
make lint
make check      # lint then test
```

## Pointers

- Product and architecture: [README.md](README.md)
- License: [LICENSE](LICENSE) (AGPLv3 or later). Marks: [TRADEMARKS.md](TRADEMARKS.md)
- Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
