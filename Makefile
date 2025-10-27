# Makefile for Go Shopping POC Application
# ------------------------------------------------------------------

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

# ------------------------------------------------------------------
# --- Load environment variables from .env and .env.local ---
# ------------------------------------------------------------------

# Verify the helper exists and is executable
ifeq (,$(wildcard ./scripts/Makefile/export_env.sh))
$(error scripts/Makefile/export_env.sh not found — please create it and make it executable)
endif

# Path for the Make-friendly export file (stable path; not ephemeral)
ENV_TMP := $(abspath .make_env_exports.mk)

# Synchronously generate the export file now (runs during parse time).
# This runs the helper and writes its output into ENV_TMP before we include it.
$(shell ./scripts/Makefile/export_env.sh > $(ENV_TMP))

# Include the generated file — it must exist because of the line above.
include $(ENV_TMP)

# Keep the generated file around so subsequent make runs can reuse it;
# if you want to force regeneration, run `make clean-env` (see target below).
.PHONY: clean-env
clean-env:
	@rm -f $(ENV_TMP)

# ----------------------------------------------------------------------
# --- This force PROJECT_HOME so that the sub-makefiles are included ---
# ----------------------------------------------------------------------
#$(info PROJECT_HOME after env load: $(PROJECT_HOME))
# Force immediate expansion so conditionals/includes see the value at parse time
# Timing issue with GNU Make: we need PROJECT_HOME to be expanded/set before we do the includes below.
PROJECT_HOME := $(PROJECT_HOME)

# ------------------------------------------------------------------
# --- Include sub-makefiles (use real paths under PROJECT_HOME) ---
# ------------------------------------------------------------------
ifneq ($(strip $(PROJECT_HOME)),)
	ifneq ($(wildcard $(PROJECT_HOME)/scripts/Makefile/postgres.mk),)
		include $(PROJECT_HOME)/scripts/Makefile/postgres.mk
	else
		$(warning $(PROJECT_HOME)/scripts/Makefile/postgres.mk not found — postgres targets not loaded)
	endif

	ifneq ($(wildcard $(PROJECT_HOME)/scripts/Makefile/kafka.mk),)
		include $(PROJECT_HOME)/scripts/Makefile/kafka.mk
	else
		$(warning $(PROJECT_HOME)/scripts/Makefile/kafka.mk not found — kafka targets not loaded)
	endif
	
	ifneq ($(wildcard $(PROJECT_HOME)/scripts/Makefile/certificates.mk),)
		include $(PROJECT_HOME)/scripts/Makefile/certificates.mk
	else
		$(warning $(PROJECT_HOME)/scripts/Makefile/certificates.mk not found — certificates targets not loaded)
	endif

	ifneq ($(wildcard $(PROJECT_HOME)/scripts/Makefile/namespaces.mk),)
		include $(PROJECT_HOME)/scripts/Makefile/namespaces.mk
	else
		$(warning $(PROJECT_HOME)/scripts/Makefile/namespaces.mk not found — namespaces targets not loaded)
	endif
else
	$(warning PROJECT_HOME not defined after loading env files)
endif
# ------------------------------------------------------------------

SERVICES := customer eventreader
MODELS := $(shell find resources/postgresql/models/ -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

# ------------------------------------------------------------------
# --- Targets ---
# ------------------------------------------------------------------

.PHONY: info all services-build services-run test lint clean \
        services-docker-build services-install services-uninstall \
		separator

info: ## Show project configuration details
	@$(MAKE) separator
	@echo
	@echo "Makefile for Go Shopping POC Application"
	@echo "----------------------------------------"
	@echo "Environment: $(ENV)"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Services: $(SERVICES)"
	@echo "Database Models: $(MODELS)"
	@echo "----------------------------------------"

separator:
	@echo
	@echo "**************************************************************"
	@echo
	
all: clean services-build services-docker-build namespaces-create postgres-install kafka-install certificates-install services-install

uninstall: services-uninstall kafka-uninstall postgres-uninstall certificates-uninstall namespaces-delete

services-build: ## Build all services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Building all services..."
	@for svc in $(SERVICES); do \
	    echo "Building $$svc..."; \
	    GOOS=linux GOARCH=amd64 go build -o bin/$$svc ./cmd/$$svc; \
	done

services-run: ## Run (locallyall services defined in SERVICES variable
	@$(MAKE) separator
	@echo "Running all services (in background)..."
	@for svc in $(SERVICES); do \
	    echo "Running $$svc with $(ENV_FILE)..."; \
	    APP_ENV=$(ENV) go run ./cmd/$$svc & \
	done

test:
	@echo "Running tests..."
	go test ./...

lint:
	@echo "Linting..."
	golangci-lint run ./...

clean:
	@echo "Cleaning binaries..."
	rm -rf bin/*

services-docker-build: ## Build and push Docker images for all services
	@$(MAKE) separator
	@echo "Building and pushing Docker images for all services..."
	@for svc in $(SERVICES); do \
	    echo "Building Docker image for $$svc..."; \
	    docker build -t localhost:5000/go-shopping-poc/$$svc:1.0 -f cmd/$$svc/Dockerfile . ; \
	    docker push localhost:5000/go-shopping-poc/$$svc:1.0; \
	    echo "Docker image for $$svc built and pushed."; \
	done

services-install: ## Deploy all services to Kubernetes
	@$(MAKE) separator
	@echo "Deploying services to Kubernetes..."
	@echo "--------------------------------------"
	@for svc in $(SERVICES); do \
	    echo "-> Deploying $$svc... as StatefulSet"; \
	    if [ ! -f deployments/kubernetes/$$svc/$$svc-deploy.yaml ]; then \
	        echo "--> Deployment file for $$svc not found, skipping..."; \
	    else \
	        kubectl apply -f deployments/kubernetes/$$svc/$$svc-deploy.yaml; \
	    fi; \
	    echo "--> $$svc deployed to Kubernetes."; \
	    echo "--------------------------------"; \
	done
	@echo "All services deployed to Kubernetes with configurations from deployments/kubernetes."
	@echo

services-uninstall: ## Uninstall all services from Kubernetes
	@$(MAKE) separator
	@echo 
	@echo "Uninstalling services from Kubernetes..."
	@echo "----------------------------------------"
	@for svc in $(SERVICES); do \
	    echo "--> Deleting $$svc..."; \
	    if [ -f deployments/kubernetes/$$svc/$$svc-deploy.yaml ]; then \
	        kubectl delete -f deployments/kubernetes/$$svc/$$svc-deploy.yaml; \
	    fi; \
	    echo "--> Uninstalled $$svc from kubernetes."; \
	    echo "-------------------------------------"; \
	done
	@echo "All services uninstalled from Kubernetes."
	@echo "----------------------------------------"

# ------------------------------------------------------------------
# --- Enhanced Help System (Grouped and Clean)
# ------------------------------------------------------------------
.PHONY: help
help:
	@$(MAKE) separator
	@echo "📘 Go Shopping POC — Make Targets"
	@echo "=================================="
	@grep -h '^[a-zA-Z0-9_.-]*:.*##' $(MAKEFILE_LIST) | \
	awk 'BEGIN { FS=": *## *"; last="" } \
	{ \
		t=$$1; d=$$2; \
		sub(/^[[:space:]]+/, "", t); sub(/[[:space:]]+$$/, "", t); \
		sub(/^[[:space:]]+/, "", d); sub(/[[:space:]]+$$/, "", d); \
		split(t, parts, "-"); group=toupper(parts[1]); \
		if (group != last) { \
			if (last != "") print ""; \
			printf "[%s]\n", group; \
			last = group; \
		} \
		printf "  %-25s %s\n", t, d; \
	}'
	@echo
	@echo "Usage: make <target>"
	@echo "Example: make postgres-install"
	@echo
