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

.PHONY: generate-ent generate-migration
generate-ent: ## Regenerate ent code from ent/schema
	go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema

generate-migration: ## Generate a versioned Atlas migration (NAME=<name>)
	$(MISE) exec -- atlas migrate diff $(NAME) \
		--dir "file://migrations/versioned" \
		--to "ent://ent/schema" \
		--dev-url "sqlite://dev?mode=memory&_fk=1"
	cp migrations/versioned/*.sql internal/repository/ent/migrations/
