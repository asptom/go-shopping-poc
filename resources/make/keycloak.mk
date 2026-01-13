# Sub-Makefile for Keycloak installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/keycloak.mk

include $(PROJECT_HOME)/resources/make/db-template.mk
SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

KEYCLOAK_REALM_FILE := $(PROJECT_HOME)/resources/keycloak/pocstore-realm.json

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------

.PHONY: keycloak-info ## Show Keycloak configuration details
keycloak-info: 
	@$(MAKE) separator
	@echo "Keycloak Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Auth Namespace: $(AUTH_NAMESPACE)"
	@echo "Keycloak Realm File: $(KEYCLOAK_REALM_FILE)"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Keycloak database secret creation
# ------------------------------------------------------------------

.PHONY: keycloak-db-secret ## Create Keycloak database secret in Kubernetes
keycloak-db-secret: 
	@$(call db_secret,keycloak,$(AUTH_NAMESPACE))

# ------------------------------------------------------------------
# Keycloak database init configmap creation
# ------------------------------------------------------------------

.PHONY: keycloak-db-configmap ## Create Keycloak database init configmap in Kubernetes
keycloak-db-configmap: keycloak-db-secret 
	@$(call db_configmap_init_sql,keycloak,$(AUTH_NAMESPACE))

# ------------------------------------------------------------------
# Load Keycloak realm configmap into Kubernetes
# ------------------------------------------------------------------
.PHONY: keycloak-realm-configmap ## Load the Keycloak realm configuration into a configmap
keycloak-realm-configmap: 
	@$(MAKE) separator
	@bash -euo pipefail -c '\
		kubectl -n $(AUTH_NAMESPACE) create configmap keycloak-realm --from-file=realm.json=$(KEYCLOAK_REALM_FILE) --dry-run=client -o yaml | kubectl apply -f -; \
		echo "Keycloak realm configmap created"; \
	'
# ------------------------------------------------------------------
# Install keycloak platform
# ------------------------------------------------------------------

.PHONY: keycloak-install ## Install Keycloak platform in Kubernetes
keycloak-install: keycloak-db-configmap keycloak-realm-configmap 
	@$(MAKE) separator
	@echo "Installing Keycloak platform in Kubernetes..."
	@kubectl apply -f deploy/k8s/platform/keycloak/db/
	@kubectl apply -f deploy/k8s/platform/keycloak/
	@kubectl rollout status statefulset/keycloak -n $(AUTH_NAMESPACE) --timeout=180s
	@echo