# Human commands. Tool versions live in mise.toml / mise.lock.
# Requires: make, mise (brew install mise)

MISE ?= mise

.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

.PHONY: help doctor tools install lock dev test lint check

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "\n  Burrow\n\n"} \
		/^[a-zA-Z0-9_-]+:.*## / { printf "  make %-10s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf "\n"

doctor: ## Show pinned toolchain
	$(MISE) run doctor

tools: ## Install bun from mise.lock
	$(MISE) install

install: tools ## Install tools and workspace deps
	$(MISE) run install

lock: ## Refresh mise.lock checksums
	$(MISE) lock

dev: ## Local agent + web (after apps exist)
	$(MISE) run dev

test: ## Run tests
	$(MISE) run test

lint: ## Lint
	$(MISE) run lint

check: lint test ## Lint then test

.PHONY: migrate migrate-dry-run
migrate: ## Apply pending schema migrations (PROFILE=<name>)
	$(MISE) exec -- go run ./packages/engine/cmd/migrate $(if $(PROFILE),--profile $(PROFILE),)

migrate-dry-run: ## Print pending migration statements without applying
	@$(MISE) exec -- go run ./packages/engine/cmd/migrate --dry-run $(if $(PROFILE),--profile $(PROFILE),)

.PHONY: generate-ent
generate-ent: ## Regenerate ent code from ent/schema
	$(MISE) exec -- go run -mod=mod entgo.io/ent/cmd/ent generate ./packages/engine/ent/schema

