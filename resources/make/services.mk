# Sub-Makefile for services installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/services.mk

include $(PROJECT_HOME)/resources/make/db-template.mk
SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

#SERVICES := customer eventreader product
#SERVICES_WITH_DB := customer product
SERVICES := customer eventreader
SERVICES_WITH_DB := customer

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------

.PHONY: services-info ## Show services configuration details
services-info: 
	@$(MAKE) separator
	@echo "Services Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Services Namespace: $(SERVICES_NAMESPACE)"
	@echo "Services with DB: $(SERVICES_WITH_DB)"
	@echo "Services: $(SERVICES)"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Base service targets
# ------------------------------------------------------------------

.PHONY: services-build  ## Build all services defined in SERVICES variable
services-build:
	@$(MAKE) separator
	@echo "Building all services..."
	@for svc in $(SERVICES); do \
	    echo "Building $$svc..."; \
	    GOOS=linux GOARCH=amd64 go build -o bin/$$svc ./cmd/$$svc; \
	done

.PHONY: services-test  ## Run tests for all services defined in SERVICES variable
services-test:
	@$(MAKE) separator
	@echo "Running tests for all services..."
	@for svc in $(SERVICES); do \
	    echo "Running tests for $$svc..."; \
	    go test ./cmd/$$svc/...; \
	done

.PHONY: services-lint ## Run linters for all services defined in SERVICES variable
services-lint:
	@$(MAKE) separator
	@echo "Running linters for all services..."
	@for svc in $(SERVICES); do \
	    echo "Running linters for $$svc..."; \
	    golangci-lint run ./cmd/$$svc/...; \
	done
	golangci-lint run ./...

.PHONY: services-clean  ## Clean up all services defined in SERVICES variable
services-clean:
	@$(MAKE) separator
	@echo "Cleaning up all services..."
	@for svc in $(SERVICES); do \
	    echo "Cleaning up $$svc..."; \
	    go clean ./cmd/$$svc/...; \
	done

.PHONY: services-build-docker ## Build and push Docker images for all services
services-build-docker:
	@$(MAKE) separator
	@echo "Building and pushing Docker images for all services..."
	@for svc in $(SERVICES); do \
	    echo "Building Docker image for $$svc..."; \
	    docker build -t localhost:5000/go-shopping-poc/$$svc:1.0 -f cmd/$$svc/Dockerfile . ; \
	    docker push localhost:5000/go-shopping-poc/$$svc:1.0; \
	    echo "Docker image for $$svc built and pushed."; \
	done

# ------------------------------------------------------------------
# Service(s) database secret creation
# ------------------------------------------------------------------

.PHONY: services-db-secret ## Create database secrets in Kubernetes for services with DBs
services-db-secret:
	@$(MAKE) separator
	@echo "Creating database secrets for services with a database..."
	@# Use make-level iteration so service name is expanded into the function argument
	@$(foreach svc,$(SERVICES_WITH_DB),$(call db_secret,$(svc),$(SERVICES_NAMESPACE)))
	
# ------------------------------------------------------------------
# Service(s) database init configmap creation
# ------------------------------------------------------------------

.PHONY: services-db-configmap ## Create database init configmap in Kubernetes
services-db-configmap: services-db-secret
	@$(MAKE) separator
	@echo "Creating database init configmaps for services with a database..."
	@# Use make-level iteration so service name is expanded into the function argument
	@$(foreach svc,$(SERVICES_WITH_DB),$(call db_configmap_init_sql,$(svc),$(SERVICES_NAMESPACE)))

# ------------------------------------------------------------------
# Service(s) database creation
# ------------------------------------------------------------------

.PHONY: services-db-create ## Perform database creation for services with DBs
services-db-create: services-db-configmap
	@$(MAKE) separator
	@echo "Creating databases for services with a database..."
	@# Use make-level iteration so service name is expanded into the function argument
	@$(foreach svc,$(SERVICES_WITH_DB),$(call db_create,$(svc),$(SERVICES_NAMESPACE)))

# ------------------------------------------------------------------
# Service(s) database migration configmap creation
# ------------------------------------------------------------------

.PHONY: services-db-migrate-configmap ## Create database migration configmaps in Kubernetes
services-db-migrate-configmap:
	@$(MAKE) separator
	@echo "Creating database migration configmaps for services with a database..."
	@# Use make-level iteration so service name is expanded into the function argument
	@$(foreach svc,$(SERVICES_WITH_DB),$(call db_configmap_migrations_sql,$(svc),$(SERVICES_NAMESPACE)))

# ------------------------------------------------------------------
# Service(s) database migration
# ------------------------------------------------------------------

.PHONY: services-db-migrate ## Perform database migration for services with DBs
services-db-migrate: services-db-migrate-configmap services-db-create
	@$(MAKE) separator
	@echo "Migrating databases for services with a database..."
	@# Use make-level iteration so service name is expanded into the function argument
	@$(foreach svc,$(SERVICES_WITH_DB),$(call db_migrate,$(svc),$(SERVICES_NAMESPACE)))

# ------------------------------------------------------------------
# Service(s) db installation
# ------------------------------------------------------------------

.PHONY: services-db-install ## Install databases for services in Kubernetes
services-db-install: services-db-migrate
	@$(MAKE) separator
	@echo "Installing all databases for services in Kubernetes..."

# ------------------------------------------------------------------
# Service(s) only installation
# ------------------------------------------------------------------

.PHONY: services-only-install ## Install only services (no DBS) in Kubernetes
services-only-install: services-build-docker
	@$(MAKE) separator
	@echo "Installing all services (but no DBs) in Kubernetes..."
	@for svc in $(SERVICES); do \
	    echo "Installing $$svc in Kubernetes..."; \
	    kubectl apply -f deploy/k8s/service/$$svc/; \
	    kubectl rollout status statefulset/$$svc -n $(SERVICES_NAMESPACE) --timeout=180s; \
	done

# ------------------------------------------------------------------
# Service(s) complete installation
# ------------------------------------------------------------------

.PHONY: services-complete-install ## Install services and databases in Kubernetes
services-complete-install: services-db-install services-only-install
	@$(MAKE) separator
	@echo "Installing all services (no DBs) in Kubernetes..."
