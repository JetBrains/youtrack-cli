BINARY := youtrack
GO ?= go
GOFMT ?= gofmt
GO_VERSION ?= $(shell $(GO) env GOVERSION)
GOLANGCI_LINT ?= GOTOOLCHAIN=$(GO_VERSION) $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%d)
LDFLAGS := -X github.com/JetBrains/youtrack-cli/internal.Version=$(VERSION) \
           -X github.com/JetBrains/youtrack-cli/internal.Commit=$(COMMIT) \
           -X github.com/JetBrains/youtrack-cli/internal.Date=$(DATE)
E2E_SCRIPT := ./scripts/e2e/test.sh

.DEFAULT_GOAL := build

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) .

fmt:
	$(GOFMT) -w .

fmt-check:
	@unformatted="$$( $(GOFMT) -l . )"; \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint:
	$(GOLANGCI_LINT) run ./...

check: fmt-check lint build unit concurrency test

unit:
	$(GO) test ./api/ ./internal/... -count=1

concurrency:
	$(GO) test -tags concurrency -run Concurrent ./internal/... -count=1

test:
	$(E2E_SCRIPT)

e2e-up:
	./scripts/e2e/up.sh

e2e-down:
	./scripts/e2e/down.sh

e2e-logs:
	./scripts/e2e/logs.sh

snapshot:
	goreleaser release --snapshot --clean --skip=publish

docs-package:
	./scripts/docs-package.sh

# Writerside build of docs/portal/ into docs-out/site/; the builder recreates OUTPUT_DIR, so it must not be the mount point.
# On Apple Silicon, Docker needs Rosetta for amd64 images.
docs-build:
	rm -rf docs-out && mkdir -p docs-out
	docker run --rm --platform linux/amd64 \
		-v "$(CURDIR)/docs/portal:/opt/sources/portal:ro" -v "$(CURDIR)/docs-out:/opt/wrs" \
		-e SOURCE_DIR=/opt/sources -e OUTPUT_DIR=/opt/wrs/site \
		-e MODULE_INSTANCE=portal/youtrack-cli -e RUNNER=other \
		jetbrains/writerside-builder:2026.10.0423

clean:
	rm -f $(BINARY)
	rm -rf dist/

.PHONY: build fmt fmt-check lint check unit concurrency test e2e-up e2e-down e2e-logs snapshot docs-package docs-build clean
