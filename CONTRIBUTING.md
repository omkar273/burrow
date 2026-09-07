# Contributing to Burrow

The first recovery loop is **not shipped**. Useful work is the Gmail loop: connect → copy → verify → search → restore a message → export. Storage adapters and tests that fail when restore would fail count. Roadmap source names (Drive, Microsoft 365, Slack, GitHub, Notion) are not connectors yet.

Please read the [code of conduct](CODE_OF_CONDUCT.md). Security issues go to [SECURITY.md](SECURITY.md), not a public issue.

## Before you write code

Open an issue first (or comment on an existing one) if you plan a design change or a new feature. That is how restic and rclone keep duplicate work out. Small fixes can be a pull request.

AI coding agents: follow [AGENTS.md](AGENTS.md). Red-green tests for production code; YAGNI; restore is the acceptance test. You, the human, still own every line you submit.

## Develop

Toolchain is [mise](https://mise.jdx.dev). `make` is the command surface.

```bash
# once: brew install mise
# echo 'eval "$(mise activate zsh)"' >> ~/.zshrc

git clone https://github.com/omkar273/burrow.git
cd burrow
make install    # mise install + bun install
make doctor
```

`make dev` waits until `apps/agent` and `apps/web` exist. `make test` and `make lint` are stubs until those apps exist.

## Pull requests

- One concern per branch.
- Link the issue.
- Keep secrets out of the tree and out of logs.
- If the change touches copy, verify, restore, or export, say how you would prove a restore still works — even if the loop is not runnable yet.
