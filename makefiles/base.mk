ifndef _include_base_mk
_include_base_mk := 1

BUILDDIR ?= build
BINDIR := $(BUILDDIR)/bin
TMPDIR := $(shell mktemp -d)

.PHONY: all help generate lint fmt test integration-test coverage bootstrap e2e-test

all: generate fmt lint integration-test

help: ## Help
	@cat $(sort $(MAKEFILE_LIST)) | grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

generate: ## Generate

format: ## Format

lint: ## Lint

test: ## Run unit tests

integration-test: ## Run integration tests

coverage: ## Collect coverage

bootstrap: ## Bootstrap environment

e2e-test: ## Run e2e tests

endif # _include_base_mk