# Sub-Makefile for Minio installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/minio.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: minio-info minio-initialize
# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
minio-info: ## Show Minio configuration details
	@$(MAKE) separator
	@echo "Minio Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Minio Namespace: $(MINIO_NAMESPACE)"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Initialize target
# ------------------------------------------------------------------
minio-initialize: ## Create required MinIO buckets
	@$(MAKE) separator
	@echo "Skipping the initialization of Minio buckets right now..."
# 	@if [ -f .env ]; then set -a; source .env; set +a; fi
# 	@: "${MINIO_USER:?MINIO_USER must be set (export or .env)}"
# 	@: "${MINIO_PASSWORD:?MINIO_PASSWORD must be set (export or .env)}"
# 	@: "${MINIO_NAMESPACE:?MINIO_NAMESPACE must be set (export or .env)}"
# 	@echo "Creating required MinIO buckets..."
# 	@bash -euo pipefail -c '\
# 		# Get MinIO pod name \
# 		MINIO_POD=$$(kubectl get pods -n "$$MINIO_NAMESPACE" -l app=minio -o jsonpath="{.items[0].metadata.name}"); \
# 		if [ -z "$$MINIO_POD" ]; then \
# 			echo "Error: MinIO pod not found"; \
# 			exit 1; \
# 		fi; \
# 		echo "Found MinIO pod: $$MINIO_POD"; \
# 		\
# 		# Create product-images bucket \
# 		echo "Creating product-images bucket..."; \
# 		kubectl exec -n "$$MINIO_NAMESPACE" "$$MINIO_POD" -- /bin/sh -c "\
# 			mc alias set local http://localhost:9000 $$MINIO_USER $$MINIO_PASSWORD && \
# 			mc mb local/product-images --ignore-existing \
# 		" && echo "Bucket product-images created successfully" || echo "Bucket product-images may already exist"; \
	