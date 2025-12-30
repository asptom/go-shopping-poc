 # Sub-Makefile for Product Loader local development
 # Include this in your top-level Makefile with:
 #   include $(PROJECT_HOME)/resources/make/product_loader.mk

 SHELL := /usr/bin/env bash
 .SHELLFLAGS := -euo pipefail -c
 .ONESHELL:

 .PHONY: product-loader-info product-loader-build product-loader-test product-loader-run

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
product-loader-info: ## Show Product Loader configuration details
	@$(MAKE) separator
	@echo "Product Loader Configuration:"
	@echo "-----------------------------"
	@echo "Project Home: $$PROJECT_HOME"
	@echo "Source Directory: ./cmd/product-loader"
	@echo "Binary Output: ./bin/product-loader"
	@echo "-----------------------------"
	@echo

# ------------------------------------------------------------------
# Build target
# ------------------------------------------------------------------
product-loader-build: ## Build the Product Loader binary locally
	@$(MAKE) separator
	@echo "Building Product Loader..."
	@echo "Building product-loader binary..."
	go build -o bin/product-loader ./cmd/product-loader
	@echo "Product Loader build complete."

# ------------------------------------------------------------------
# Test target
# ------------------------------------------------------------------
product-loader-test: product-loader-build ## Run tests for Product Loader
	@$(MAKE) separator
	@echo "Running Product Loader tests..."
	go test ./cmd/product-loader/...
	@echo "Product Loader tests complete."


# ------------------------------------------------------------------
# Run target (for local development)
# ------------------------------------------------------------------
product-loader-run: ## Run Product Loader locally
	@$(MAKE) separator
	@echo "Running Product Loader locally..."
	@if [ -f .env ]; then \
		set -a; \
		source .env; \
		set +a; \
	fi && \
	if [ -f .env.local ]; then \
		set -a; \
		source .env.local; \
		set +a; \
	fi && \
	go run ./cmd/product-loader
	@echo "Product Loader run complete."