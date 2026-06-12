# Makefile for pct. Run `make help` for a list of targets.

BINARY := pct
GO     ?= go
LINTER ?= golangci-lint

.DEFAULT_GOAL := build

.PHONY: build
build: ## Build the pct binary
	$(GO) build -o $(BINARY) .

.PHONY: install
install: ## Install pct into GOBIN
	$(GO) install .

.PHONY: test
test: ## Run all tests
	$(GO) test ./...

.PHONY: fuzz
fuzz: ## Fuzz the expression evaluator for a short while
	$(GO) test -fuzz=FuzzEvaluate -fuzztime=30s ./internal/expr/

.PHONY: cover
cover: ## Run tests and report coverage
	$(GO) test -coverprofile=cover.out ./...
	$(GO) tool cover -func=cover.out

.PHONY: fmt
fmt: ## Reformat all sources with gofmt
	gofmt -s -w .

.PHONY: fmt-check
fmt-check: ## Fail if any source file is not gofmt-formatted
	@unformatted=$$(gofmt -s -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@command -v $(LINTER) >/dev/null || { \
		echo "$(LINTER) not found; see https://golangci-lint.run/welcome/install/"; exit 1; }
	$(LINTER) run

.PHONY: check
check: fmt-check vet lint test ## Run every quality gate

.PHONY: clean
clean: ## Remove build and coverage artifacts
	rm -f $(BINARY) cover.out

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | \
		awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'
