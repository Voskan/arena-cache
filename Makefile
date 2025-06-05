# arena-cache – project root Makefile
# ------------------------------------
# Helpful aliases for local development; CI does *not* rely on Make, all steps
# are duplicated explicitly in GitHub Actions workflows.  Targets are ordered
# by common usage.

GO           ?= go
GOFLAGS      ?=
MODULE       := $(shell $(GO) list -m)

BIN_DIR      := ./bin
BIN_INSPECT  := $(BIN_DIR)/arena-cache-inspect

.PHONY: all dev lint test bench tidy docs docs-serve docker clean tools install-tools

all: lint test

## dev: Run examples/basic locally
.PHONY: dev
dev:
	$(GO) run ./examples/basic

## lint: Run go vet + staticcheck
lint:
	@echo "\n→ go vet"
	$(GO) vet ./...
	@echo "\n→ staticcheck"
	$(GO) run honnef.co/go/tools/cmd/staticcheck@latest ./...

## test: Run unit tests with race detector & coverage
TEST_PKGS := $(shell $(GO) list ./... | grep -v "/bench$")

test:
	$(GO) test $(GOFLAGS) -race -cover -coverprofile=coverage.out $(TEST_PKGS)
	@echo "coverage saved to coverage.out"

## bench: Run micro-benchmarks
bench:
	$(GO) test ./bench -bench=. -benchmem -count=3 | tee bench.txt

## tidy: Ensure go.{mod,sum} are tidy

tidy:
	$(GO) mod tidy

## docs: Build docs via MkDocs

docs:
	mkdocs build --strict

## docs-serve: Live preview of docs

docs-serve:
	mkdocs serve -a 127.0.0.1:8000

## docker: Build local development image for inspector

docker:
	DOCKER_BUILDKIT=1 docker build -f build/docker/Dockerfile \
	  --build-arg VERSION=dev -t arena-cache-inspect:dev .

## tools: Install developer tools into GOPATH/bin

TOOLS := \
  golang.org/x/perf/cmd/benchstat@latest \
  honnef.co/go/tools/cmd/staticcheck@latest

install-tools:
	@for t in $(TOOLS); do \
	  echo "Installing $$t"; \
	  $(GO) install $$t; \
	done

## clean: Remove generated files
clean:
	rm -f coverage.out bench.txt
	rm -rf $(BIN_DIR)

# Default target when none specified
.DEFAULT_GOAL := all
