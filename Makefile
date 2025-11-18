
SWAG_CMD ?= swag
GO ?= go
BINARY ?= obsync
OUTDIR ?= bin

.PHONY: all install-swag docs build run tidy fmt vet test clean docker-build

all: build

install-swag:
	@echo "Install swag CLI into \\$(GOBIN) or \\$(GOPATH)/bin if not set"
	@echo "Run: go install github.com/swaggo/swag/cmd/swag@latest"

docs: ## Generate swagger docs using swag (requires `swag` in PATH)
	@echo "Generating swagger docs..."
	$(SWAG_CMD) init -g main.go -o docs
	@# keep only embedded docs (docs/docs.go); remove generated static artifacts
	@if [ -f docs/swagger.json ]; then rm -f docs/swagger.json; fi
	@if [ -f docs/swagger.yaml ]; then rm -f docs/swagger.yaml; fi
	@if [ -f docs/openapi.json ]; then rm -f docs/openapi.json; fi

build: docs tidy fmt vet ## Build the project binary (generates docs first)
	@mkdir -p $(OUTDIR)
	$(GO) build -o $(OUTDIR)/$(BINARY) main.go

run: build ## Build then run
	./$(OUTDIR)/$(BINARY)

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(OUTDIR)
	$(GO) clean

docker-build: ## Build a Docker image (requires Dockerfile)
	docker build -t obsync:latest .

