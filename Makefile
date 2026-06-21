BINARY_NAME=postgrebase
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
PRE_VERSION=$(if $(filter %-pre,$(VERSION)),$(VERSION),$(VERSION)-pre)
LDFLAGS=-ldflags "-s -w -X github.com/zhenruyan/postgrebase.Version=$(VERSION)"
GOBUILD_FLAGS=-trimpath
BIN_DIR=bin
DIST_DIR=dist
SDK_DIR=js-sdk

# UPX compression (skip unsupported architectures)
USE_UPX ?= true
ifeq ($(shell which upx 2>/dev/null),)
USE_UPX = false
endif
ifeq ($(USE_UPX),true)
UPX_CMD = upx -9
else
UPX_CMD = @true
endif

.PHONY: help build build-all clean lint fmt run
.PHONY: build-linux build-linux-musl build-darwin build-windows
.PHONY: npm-version npm-packages npm-pack npm-publish-all npm-publish-pre
.PHONY: sdk-build sdk-test sdk-pack sdk-publish sdk-publish-pre

# Default target
help:
	@echo "PostgreBase Build System"
	@echo ""
	@echo "Build targets:"
	@echo "  build            Build for current platform"
	@echo "  build-linux      Build for Linux (amd64, arm64)"
	@echo "  build-linux-musl Build for Linux musl (amd64)"
	@echo "  build-darwin     Build for macOS (amd64, arm64)"
	@echo "  build-windows    Build for Windows (amd64, arm64)"
	@echo "  build-all        Build for all platforms and architectures"
	@echo ""
	@echo "NPM Installer targets:"
	@echo "  npm-version       Sync version to npm installer package"
	@echo "  npm-packages      Build platform-specific npm packages"
	@echo "  npm-pack          Pack main + all platform packages"
	@echo "  npm-publish-all   Publish main + all platform packages"
	@echo "  npm-publish-pre   Publish pre-release packages"
	@echo ""
	@echo "JS SDK targets:"
	@echo "  sdk-build         Build the JavaScript SDK"
	@echo "  sdk-test          Run JS SDK tests"
	@echo "  sdk-pack          Pack JS SDK for publishing"
	@echo "  sdk-publish       Publish JS SDK to npm"
	@echo "  sdk-publish-pre   Publish JS SDK pre-release"
	@echo ""
	@echo "Other targets:"
	@echo "  lint              Run linter"
	@echo "  fmt               Format code"
	@echo "  clean             Remove build artifacts"
	@echo "  run               Build and run"
	@echo "  help              Show this help"

# Build for current platform
build:
	CGO_ENABLED=0 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./build/

# Platform builds
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 ./build/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 ./build/
	@echo "Compressing Linux binaries with UPX..."
	$(UPX_CMD) $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 || true
	$(UPX_CMD) $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 || true

build-linux-musl:
	@echo "Building for Linux musl..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux-musl-amd64 ./build/
	@echo "Compressing Linux musl binary with UPX..."
	$(UPX_CMD) $(BIN_DIR)/$(BINARY_NAME)-linux-musl-amd64 || true

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 ./build/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 ./build/
	@echo "Compressing macOS amd64 binary with UPX (arm64 not supported)..."
	$(UPX_CMD) $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 || true

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe ./build/
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(GOBUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe ./build/
	@echo "Compressing Windows amd64 binary with UPX (arm64 not supported)..."
	$(UPX_CMD) $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe || true

# Build all platforms
build-all: build-linux build-linux-musl build-darwin build-windows
	@echo ""
	@echo "Build complete! Binaries in $(BIN_DIR)/"
	@ls -lh $(BIN_DIR)/

# Lint
lint:
	golangci-lint run -c ./golangci.yml ./...

# Format
fmt:
	gofmt -w .

# Clean
clean:
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)
	rm -f npm/*.tgz
	rm -rf $(SDK_DIR)/dist
	rm -rf $(SDK_DIR)/node_modules

# Run
run: build
	./$(BIN_DIR)/$(BINARY_NAME) serve --dataDsn "sqlite://./pb_data/dev.db"

# ============================================================================
# NPM Installer targets
# ============================================================================

npm-version:
	./scripts/sync-npm-version.sh $(VERSION)

# Build platform-specific npm packages
npm-packages: build-all
	./scripts/build-npm-packages.sh

# Pack main + platform packages
npm-pack: npm-version npm-packages
	@echo "Packing platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Packing $$(basename $$d)..."; \
			cd "$$d" && npm pack && cd - > /dev/null; \
			mv "$$d"/*.tgz npm/ 2>/dev/null || true; \
		fi; \
	done
	@echo "Packing main package..."
	cd npm && npm pack
	@echo "Done. Tarballs in npm/"

# Publish platform packages first, then main
npm-publish-all: npm-version npm-packages
	@echo "Publishing platform packages..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Publishing $$(basename $$d)..."; \
			cd "$$d" && npm publish --tag latest && cd - > /dev/null; \
		fi; \
	done
	@echo "Publishing main package..."
	cd npm && npm publish --tag latest
	@echo "Published all packages!"

# Publish pre-release
npm-publish-pre:
	./scripts/sync-npm-version.sh $(PRE_VERSION)
	$(MAKE) npm-packages VERSION=$(PRE_VERSION)
	@echo "Publishing platform packages (pre-release)..."
	@for d in npm/packages/*/; do \
		if [ -f "$$d/package.json" ]; then \
			echo "  Publishing $$(basename $$d)..."; \
			cd "$$d" && npm publish --tag next && cd - > /dev/null; \
		fi; \
	done
	@echo "Publishing main package (pre-release)..."
	cd npm && npm publish --tag next
	@echo "Published all packages (pre-release)!"

# ============================================================================
# JS SDK targets
# ============================================================================

# Build JS SDK
sdk-build:
	@echo "Building JS SDK..."
	cd $(SDK_DIR) && npm install && npm run build
	@echo "JS SDK built successfully!"

# Run JS SDK tests
sdk-test:
	@echo "Running JS SDK tests..."
	cd $(SDK_DIR) && npm test
	@echo "JS SDK tests complete!"

# Pack JS SDK
sdk-pack: sdk-build
	@echo "Packing JS SDK..."
	cd $(SDK_DIR) && npm pack
	@echo "JS SDK packed!"

# Publish JS SDK
sdk-publish: sdk-build
	@echo "Publishing JS SDK..."
	cd $(SDK_DIR) && npm publish --tag latest
	@echo "JS SDK published!"

# Publish JS SDK pre-release
sdk-publish-pre:
	@echo "Publishing JS SDK (pre-release)..."
	cd $(SDK_DIR) && npm version $(PRE_VERSION) --no-git-tag-version
	cd $(SDK_DIR) && npm publish --tag next
	@echo "JS SDK published (pre-release)!"
