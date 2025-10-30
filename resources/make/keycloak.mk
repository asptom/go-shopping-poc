# Sub-Makefile for TLS certificate installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/certificates.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: keycloak-info keycloak-realm keycloak-install keycloak-uninstall

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
keycloak-info: ## Show Keycloak configuration details
	@$(MAKE) separator
	@echo "Keycloak Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $$PROJECT_HOME"
	@echo "Keycloak Namespace: $$KEYCLOAK_NAMESPACE"
	@echo "Resource Dir: $$KEYCLOAK_RESOURCES_DIR"
	@echo "Deployment Dir: $$KEYCLOAK_DEPLOYMENT_DIR"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Load realm into configmap
# ------------------------------------------------------------------
keycloak-realm: ## Load the Keycloak realm configuration into a configmap
	@$(MAKE) separator
	@[ -d "$$KEYCLOAK_RESOURCES_DIR" ] || { echo "Resources directory missing: $$KEYCLOAK_RESOURCES_DIR"; exit 1; }
	@bash -euo pipefail -c '\
		kubectl -n "$$KEYCLOAK_NAMESPACE" create configmap keycloak-realm --from-file=realm.json="$$KEYCLOAK_RESOURCES_DIR/pocstore-realm.json" --dry-run=client -o yaml | kubectl apply -f -; \
		echo "Keycloak realm configmap created"; \
	'

# ------------------------------------------------------------------
# Install target
# ------------------------------------------------------------------
keycloak-install: ## Deploy Keycloak into Kubernetes
	@$(MAKE) separator
	@[ -d "$$KEYCLOAK_DEPLOYMENT_DIR" ] || { echo "Deployment directory missing: $$KEYCLOAK_DEPLOYMENT_DIR"; exit 1; }
	@$(MAKE) keycloak-realm
	@bash -euo pipefail -c '\
		kubectl -n "$$KEYCLOAK_NAMESPACE" apply -f "$$KEYCLOAK_DEPLOYMENT_DIR/keycloak-deploy.yaml"; \
		echo "Keycloak deployed"; \
	'
# ------------------------------------------------------------------
# Uninstall target
# ------------------------------------------------------------------
keycloak-uninstall: ## Delete Keycloak from Kubernetes
	@$(MAKE) separator
	@[ -d "$$KEYCLOAK_DEPLOYMENT_DIR" ] || { echo "Deployment directory missing: $$KEYCLOAK_DEPLOYMENT_DIR"; exit 1; }
	@bash -euo pipefail -c '\
		kubectl -n "$$KEYCLOAK_NAMESPACE" delete -f "$$KEYCLOAK_DEPLOYMENT_DIR/keycloak-deploy.yaml"; \
		echo "Keycloak uninstalled"; \
	'