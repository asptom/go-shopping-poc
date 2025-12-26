# Sub-Makefile for Minio installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/certificates.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: minio-info minio-secret minio-install minio-uninstall minio-create-buckets

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
minio-info: ## Show Minio configuration details
	@$(MAKE) separator
	@echo "Minio Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $$PROJECT_HOME"
	@echo "Minio Namespace: $$MINIO_NAMESPACE"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Load secret into Kubernetes
# ------------------------------------------------------------------
minio-secret: ## Load the Minio secret into Kubernetes
	@$(MAKE) separator
	@if [ -f .env ]; then set -a; source .env; set +a; fi
	@# Fail early if required values missing
	@: "${MINIO_USER:?MINIO_USER must be set (export or .env)}"
	@: "${MINIO_PASSWORD:?MINIO_PASSWORD must be set (export or .env)}"
	@: "${MINIO_NAMESPACE:?MINIO_NAMESPACE must be set (export or .env)}"
	@printf "%s\n" \
	    "apiVersion: v1" \
		"kind: Secret" \
        "metadata:" \
        "  name: minio-credentials" \
        "  namespace: $$MINIO_NAMESPACE" \
        "type: Opaque" \
        "stringData:" \
        "  MINIO_USER: $$MINIO_USER" \
        "  MINIO_PASSWORD: $$MINIO_PASSWORD" \
    | kubectl apply -f - ;
	@echo "Secret 'minio-credentials' applied to namespace $$MINIO_NAMESPACE"

# ------------------------------------------------------------------
# Install target
# ------------------------------------------------------------------
minio-install: ## Deploy Minio into Kubernetes
	@$(MAKE) separator
	@[ -d "$$MINIO_DEPLOYMENT_DIR" ] || { echo "Deployment directory missing: $$MINIO_DEPLOYMENT_DIR"; exit 1; }
	@$(MAKE) minio-secret
	@bash -euo pipefail -c '\
		kubectl -n "$$MINIO_NAMESPACE" apply -f "$$MINIO_DEPLOYMENT_DIR/minio-deploy.yaml"; \
		echo "Minio deployed"; \
	'
	@echo "Waiting for MinIO to be ready..."
	@kubectl wait --for=condition=ready pod -l app=minio -n "$$MINIO_NAMESPACE" --timeout=300s
	@$(MAKE) minio-create-buckets
# ------------------------------------------------------------------
# Create buckets target
# ------------------------------------------------------------------
minio-create-buckets: ## Create required MinIO buckets
	@$(MAKE) separator
	@if [ -f .env ]; then set -a; source .env; set +a; fi
	@: "${MINIO_USER:?MINIO_USER must be set (export or .env)}"
	@: "${MINIO_PASSWORD:?MINIO_PASSWORD must be set (export or .env)}"
	@: "${MINIO_NAMESPACE:?MINIO_NAMESPACE must be set (export or .env)}"
	@echo "Creating required MinIO buckets..."
	@bash -euo pipefail -c '\
		# Get MinIO pod name \
		MINIO_POD=$$(kubectl get pods -n "$$MINIO_NAMESPACE" -l app=minio -o jsonpath="{.items[0].metadata.name}"); \
		if [ -z "$$MINIO_POD" ]; then \
			echo "Error: MinIO pod not found"; \
			exit 1; \
		fi; \
		echo "Found MinIO pod: $$MINIO_POD"; \
		\
		# Create product-images bucket \
		echo "Creating product-images bucket..."; \
		kubectl exec -n "$$MINIO_NAMESPACE" "$$MINIO_POD" -- /bin/sh -c "\
			mc alias set local http://localhost:9000 $$MINIO_USER $$MINIO_PASSWORD && \
			mc mb local/product-images --ignore-existing \
		" && echo "Bucket product-images created successfully" || echo "Bucket product-images may already exist"; \
	'
# ------------------------------------------------------------------
# Uninstall target
# ------------------------------------------------------------------
minio-uninstall: ## Delete Minio from Kubernetes
	@$(MAKE) separator
	@[ -d "$$MINIO_DEPLOYMENT_DIR" ] || { echo "Deployment directory missing: $$MINIO_DEPLOYMENT_DIR"; exit 1; }
	@bash -euo pipefail -c '\
		kubectl -n "$$MINIO_NAMESPACE" delete -f "$$MINIO_DEPLOYMENT_DIR/minio-deploy.yaml"; \
		echo "Minio uninstalled"; \
	'