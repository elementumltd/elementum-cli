default: build

# Go binary (adjust if not in PATH)
GO ?= go
GOBIN := $(shell $(GO) env GOPATH)/bin

# Platform detection (GNU Make directive — evaluated at parse time, shell-agnostic)
BINARY_NAME := ei
ifeq ($(OS),Windows_NT)
    WINDOWS := true
    BINARY_NAME := ei.exe
else
    WINDOWS :=
endif

build:
	${GO} build -o $(GOBIN)/$(BINARY_NAME) -v .

clean:
	rm -f $(GOBIN)/$(BINARY_NAME)

fmt:
	gofmt -s -w -e .

tidy:
	${GO} vet ./...
	${GO} mod tidy

# Lint (requires golangci-lint)
# Install from https://golangci-lint.run/docs/welcome/install/local/
# curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.11.4
lint:
	$(GOBIN)/golangci-lint run -v

# Run only CLI tests
test:
	@echo "Running CLI tests..."
	${GO} test -v -timeout=30s ./...
	${GO} test -v -timeout=30s ./internal/fieldtypes/...

# Generate GraphQL client code from schema
# Validate GraphQL operations against schema
generate-graphql:
	@echo "Generating GraphQL client code..."
	cd internal/client && ${GO} generate
	@echo "✓ GraphQL client code generated (internal/client/generated.go)"
	@echo "Validating GraphQL operations compile..."
	${GO} build ./internal/client/...
	@echo "✓ GraphQL operations valid"

# Install skills to all known agent skill directories
skills-install:
	@echo "Installing Elementum skills..."
	@for dir in ~/.claude/skills ~/.agents/skills ~/.cursor/skills; do \
		mkdir -p "$$dir"; \
		rm -rf "$$dir"/elementum-cli "$$dir"/README.md 2>/dev/null || true; \
		cp -r skills/elementum-cli "$$dir/"; \
		cp skills/README.md "$$dir/" 2>/dev/null || true; \
		echo "  ✓ $$dir"; \
	done
	@echo ""
	@echo "Installed skills:"
	@ls -1 ~/.claude/skills/ | grep elementum | sed 's/^/    /'

# Package skills as ZIP files for distribution
skills-package:
	@echo "Packaging skills..."
	@mkdir -p dist/skills
	@for skill in skills/*/; do \
		if [ -d "$$skill" ]; then \
			name=$$(basename $$skill); \
			cd $$skill && zip -r ../../dist/skills/$$name.zip . && cd ->/dev/null; \
			echo "  ✓ dist/skills/$$name.zip"; \
		fi \
	done

# Show help
help:
	@echo "Elementum CLI"
	@echo ""
	@echo "Development:"
	@echo "  build             - Build the ei CLI (default)"
	@echo "  clean             - Remove built binaries"
	@echo "  fmt               - Format GO code"
	@echo "  lint              - Run linter"
	@echo "  tidy              - Tidy go modules"
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run all tests (provider + CLI)"
	@echo ""
	@echo "Claude Code:"
	@echo "  skills-install    - Install skills to ~/.claude/skills/, ~/.agents/skills/, ~/.cursor/skills/"
	@echo "  skills-package    - Package skills as ZIP files in dist/skills/"
