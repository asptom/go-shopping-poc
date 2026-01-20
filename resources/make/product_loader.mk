 # Sub-Makefile for Product Loader local development
 # Include this in your top-level Makefile with:
 #   include $(PROJECT_HOME)/resources/make/product_loader.mk

 
# ------------------------------------------------------------------
# Build target
# ------------------------------------------------------------------
.PHONY: product-loader-build ## Build the Product Loader binary locally
product-loader-build:
	@$(MAKE) separator
	@echo "Building Product Loader..."
	@echo "Building product-loader binary..."
	go build -o bin/product-loader ./cmd/product-loader
	@echo "Product Loader build complete."

# ------------------------------------------------------------------
# Test target
# ------------------------------------------------------------------
.PHONY: product-loader-test ## Run tests for Product Loader
product-loader-test: product-loader-build
	@$(MAKE) separator
	@echo "Running Product Loader tests..."
	go test ./cmd/product-loader/...
	@echo "Product Loader tests complete."


# ------------------------------------------------------------------
# Run target (for local development)
# ------------------------------------------------------------------
.PHONY: product-loader-run ## Run Product Loader locally
product-loader-run:
	@$(MAKE) separator
	@echo "Running Product Loader locally..."
	go run ./cmd/product-loader
	@echo "Product Loader run complete."