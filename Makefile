# ==============================================================================
# uvm (Universal Version Manager) - Makefile
# ==============================================================================

BINARY_NAME=uvm
VERSION?=0.0.4
DIST_DIR=dist
BIN_DIR=bin
INSTALL_DIR?=$(HOME)/.uvm/bin

.PHONY: all build test test-coverage install uninstall cross-compile package clean help

all: test build

## build: Compile local CLI binary for current platform
build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME) main.go
	@echo "Built $(BIN_DIR)/$(BINARY_NAME)"

## test: Run unit tests across all packages
test:
	go test -v ./...

## test-coverage: Run unit tests and enforce statement code coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## install: Install binary to user environment ($INSTALL_DIR)
install: build
	@mkdir -p $(INSTALL_DIR)
	cp -f $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)/$(BINARY_NAME)"

## uninstall: Remove binary from user environment
uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from $(INSTALL_DIR)"

## cross-compile: Cross-compile CLI & Visual Installers for macOS, Linux, and Windows
cross-compile:
	@chmod +x ./scripts/build.sh
	./scripts/build.sh $(VERSION)

## package: Package release artifacts with checksums
package: cross-compile

## clean: Clean build artifacts and temporary files
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out coverage.html

## help: Display this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
