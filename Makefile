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

.PHONY: clean clean-archive
clean: ## Remove build artifacts and dry-run dumps from the repo
	@rm -f burrow migrate ./*.sql
	@$(MISE) exec -- go clean -testcache
	@printf '  cleaned build artifacts\n'

# Deliberately not part of `clean`: this deletes an archive, and once Gmail
# sync exists that is somebody's only independent copy of their mail.
clean-archive: ## DESTRUCTIVE. Delete the local archive (PROFILE=<name> for one)
	@root="$$HOME/.burrow"; \
	if [ -n "$(PROFILE)" ]; then root="$$HOME/.burrow/profiles/$(PROFILE)"; fi; \
	if [ ! -d "$$root" ]; then printf '  nothing at %s\n' "$$root"; exit 0; fi; \
	printf '\n  About to delete: %s\n\n' "$$root"; \
	for db in $$(find "$$root" -name state.db 2>/dev/null); do \
	  profile=$$(basename $$(dirname "$$db")); \
	  objects=$$(find "$$(dirname $$db)/objects" -type f 2>/dev/null | wc -l | tr -d " "); \
	  printf '    profile %-12s %s objects\n' "$$profile" "$$objects"; \
	done; \
	printf '    %s on disk\n\n' "$$(du -sh "$$root" 2>/dev/null | cut -f1)"; \
	if [ "$(CONFIRM)" = "yes" ]; then \
	  rm -rf "$$root"; printf '  deleted (CONFIRM=yes)\n'; \
	else \
	  printf '  Re-run with CONFIRM=yes to delete:\n    make clean-archive CONFIRM=yes\n'; \
	fi

.PHONY: generate-ent
generate-ent: ## Regenerate ent code from ent/schema
	$(MISE) exec -- go run -mod=mod entgo.io/ent/cmd/ent generate ./packages/engine/ent/schema

