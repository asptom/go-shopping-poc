# Sub-Makefile for project namespace creation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/namespace.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: namespaces-info namespaces-create namespaces-delete

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
namespaces-info: ## Show namespace configuration details
	@$(MAKE) separator
	@echo "Namespace Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $$PROJECT_HOME"
	@echo "Project Namespace: $$PROJECT_NAMESPACE"
	@echo "Postgresql Namespace: $$PSQL_NAMESPACE"
	@echo "Kafka Namespace: $$KAFKA_CLUSTER_NAMESPACE"
	@echo "Keycloak Namespace: $$KEYCLOAK_NAMESPACE"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Create target
# ------------------------------------------------------------------
namespaces-create: ## Create all necessary namespaces
	@$(MAKE) separator
	@echo "Creating namespaces..."
	@kubectl create namespace $$PROJECT_NAMESPACE || true
	@kubectl create namespace $$PSQL_NAMESPACE || true
	@kubectl create namespace $$KAFKA_CLUSTER_NAMESPACE || true
	@kubectl create namespace $$KEYCLOAK_NAMESPACE || true
	@echo "Namespaces created (if they did not already exist)."

# ------------------------------------------------------------------
# Delete target
# ------------------------------------------------------------------
namespaces-delete: ## Delete all created namespaces
	@$(MAKE) separator
	@echo "Deleting namespaces..."
	@if kubectl get namespace "$$PROJECT_NAMESPACE" >/dev/null 2>&1; then \
        kubectl delete namespace "$$PROJECT_NAMESPACE"; \
    else \
        echo "Namespace $$PROJECT_NAMESPACE not found, skipping."; \
    fi
	@if kubectl get namespace "$$PSQL_NAMESPACE" >/dev/null 2>&1; then \
        kubectl delete namespace "$$PSQL_NAMESPACE"; \
    else \
        echo "Namespace $$PSQL_NAMESPACE not found, skipping."; \
    fi
	@if kubectl get namespace "$$KAFKA_CLUSTER_NAMESPACE" >/dev/null 2>&1; then \
        kubectl delete namespace "$$KAFKA_CLUSTER_NAMESPACE"; \
    else \
        echo "Namespace $$KAFKA_CLUSTER_NAMESPACE not found, skipping."; \
    fi
	@if kubectl get namespace "$$KEYCLOAK_NAMESPACE" >/dev/null 2>&1; then \
        kubectl delete namespace "$$KEYCLOAK_NAMESPACE"; \
    else \
        echo "Namespace $$KEYCLOAK_NAMESPACE not found, skipping."; \
    fi
	@echo "Namespaces deleted (if they existed)."