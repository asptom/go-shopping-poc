# ------------------------------------------------------------------
# Makefile for Go Shopping POC Application
# ------------------------------------------------------------------

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

PROJECT_HOME := /Users/tom/Projects/Go/go-shopping-poc

# ------------------------------------------------------------------
# --- Include sub-makefiles (use real paths under PROJECT_HOME) ---
# ------------------------------------------------------------------
ifneq ($(strip $(PROJECT_HOME)),)

#Postgres Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/postgres.mk),)
	include $(PROJECT_HOME)/resources/make/postgres.mk
else
	$(warning $(PROJECT_HOME)/resources/make/postgres.mk not found — postgres targets not loaded)
endif

# Kafka Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/kafka.mk),)
	include $(PROJECT_HOME)/resources/make/kafka.mk
else
	$(warning $(PROJECT_HOME)/resources/make/kafka.mk not found — kafka targets not loaded)
endif

#Certificates Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/certificates.mk),)
	include $(PROJECT_HOME)/resources/make/certificates.mk
else
	$(warning $(PROJECT_HOME)/resources/make/certificates.mk not found — certificates targets not loaded)
endif

# Minio Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/minio.mk),)
	include $(PROJECT_HOME)/resources/make/minio.mk
else
	$(warning $(PROJECT_HOME)/resources/make/minio.mk not found — minio targets not loaded)
endif

# Kubernetes Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/k8s.mk),)
	include $(PROJECT_HOME)/resources/make/k8s.mk
else
	$(warning $(PROJECT_HOME)/resources/make/k8s.mk not found — kubernetes targets not loaded)
endif

# Keycloak Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/keycloak.mk),)
	include $(PROJECT_HOME)/resources/make/keycloak.mk
else
	$(warning $(PROJECT_HOME)/resources/make/keycloak.mk not found — keycloak targets not loaded)
endif

# Product Loader Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/product_loader.mk),)
	include $(PROJECT_HOME)/resources/make/product_loader.mk
else
	$(warning $(PROJECT_HOME)/resources/make/product_loader.mk not found — product_loader targets not loaded)
endif

else
	$(warning PROJECT_HOME not defined after loading env files)
endif
# ------------------------------------------------------------------

SERVICES := customer eventreader product
SERVICES_WITH_DB := customer product
SERVICES_NAMESPACE := shopping
AUTH_NAMESPACE := keycloak
DB_NAMESPACE := postgres
PLATFORMS_WITH_DB := keycloak

# ------------------------------------------------------------------
# --- Targets ---
# ------------------------------------------------------------------

.PHONY: info ## Show project configuration details
info: 
	@$(MAKE) separator
	@echo
	@echo "Makefile for Go Shopping POC Application"
	@echo "----------------------------------------"
	@echo "Environment: $(ENV)"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Services with a DB: $(SERVICES_WITH_DB)"
	@echo "Platforms with a DB: $(PLATFORMS_WITH_DB)"
	@echo "Services Namespace: $(SERVICES_NAMESPACE)"
	@echo "----------------------------------------"

.PHONY: separator
separator:
	@echo
	@echo "**************************************************************"
	@echo

.PHONY: platform ## Install all platform services (Keycloak, Postgres, Minio, Kafka, Certificates)
platform: k8s-namespaces certificates-install postgres-install keycloak-install

install: services-clean services-build services-docker-build k8s-install \
		certificates-install postgres-initialize kafka-initialize minio-initialize k8s-install-domain-services ## Full setup: build and deploy all components

uninstall: k8s-uninstall ## Uninstall all services and all supporting components

services-build: ## Build all services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Building all services..."
	@for svc in $(SERVICES); do \
	    echo "Building $$svc..."; \
	    GOOS=linux GOARCH=amd64 go build -o bin/$$svc ./cmd/$$svc; \
	done

services-run: ## Run (locally) all services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Running all services (in background)..."
	@for svc in $(SERVICES); do \
	    echo "Running $$svc with $(ENV_FILE)..."; \
	    APP_ENV=$(ENV) go run ./cmd/$$svc & \
	done

services-test: ## Run tests for all services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Running tests for all services..."
	@for svc in $(SERVICES); do \
	    echo "Running tests for $$svc..."; \
	    go test ./cmd/$$svc/...; \
	done

services-lint: ## Run linters for all services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Running linters for all services..."
	@for svc in $(SERVICES); do \
	    echo "Running linters for $$svc..."; \
	    golangci-lint run ./cmd/$$svc/...; \
	done
	golangci-lint run ./...

services-clean: ## Clean up all services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Cleaning up all services..."
	@for svc in $(SERVICES); do \
	    echo "Cleaning up $$svc..."; \
	    go clean ./cmd/$$svc/...; \
	done

services-docker-build: ## Build and push Docker images for all services
	@$(MAKE) separator
	@echo "Building and pushing Docker images for all services..."
	@for svc in $(SERVICES); do \
	    echo "Building Docker image for $$svc..."; \
	    docker build -t localhost:5000/go-shopping-poc/$$svc:1.0 -f cmd/$$svc/Dockerfile . ; \
	    docker push localhost:5000/go-shopping-poc/$$svc:1.0; \
	    echo "Docker image for $$svc built and pushed."; \
	done

# ------------------------------------------------------------------
# --- Enhanced Help System (Grouped and Clean)
# ------------------------------------------------------------------
.PHONY: help
help:
	@$(MAKE) separator
	@echo "📘 Go Shopping POC — Make Targets"
	@echo "=================================="
	@echo
	@echo "install			Full setup: clean, build, docker-build, create namespaces, install postgres, kafka, certificates, keycloak, minio, and services"
	@echo "uninstall		Uninstall all services, and all supporting components"
	@echo
	@./resources/make/make-help.sh || true
	@echo
	@echo "Usage: make <target>"
	@echo "Example: make postgres-install"
	@$(MAKE) separator
