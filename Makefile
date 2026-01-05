# Makefile for Go Shopping POC Application
# ------------------------------------------------------------------

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

# ------------------------------------------------------------------
# --- Load environment variables from .env and .env.local ---
# ------------------------------------------------------------------

# Verify the helper exists and is executable
ifeq (,$(wildcard ./resources/make/export_env.sh))
$(error ./resources/make/export_env.sh not found — please create it and make it executable)
endif

# Path for the Make-friendly export file (stable path; not ephemeral)
ENV_TMP := $(abspath ./resources/make/.make_env_exports.mk)

# Synchronously generate the export file now (runs during parse time).
# This runs the helper and writes its output into ENV_TMP before we include it.
$(shell ./resources/make/export_env.sh > $(ENV_TMP))

# Include the generated file — it must exist because of the line above.
include $(ENV_TMP)

# Keep the generated file around so subsequent make runs can reuse it;
# if you want to force regeneration, run `make env-clean` (see target below).
.PHONY: env-clean
env-clean:  ## Remove the temporary environment export file to force regeneration on next make run
	@echo "Removing temporary environment export file: $(ENV_TMP)"
	@rm -f $(ENV_TMP)

# Compare env files
.PHONY: env-diff
env-diff:  ## Compare .env and .env.local for differences
	@echo ""
	@bash -euo pipefail -c '\
		echo "----------------------------------------------"; \
		echo "Comparing .env and .env.local..."; \
		ENV_SORTED=$$(mktemp); \
		ENV_LOCAL_SORTED=$$(mktemp); \
		grep -v "^\s*#" .env | grep -v "^\s*$$" | sort > $$ENV_SORTED; \
		grep -v "^\s*#" .env.local | grep -v "^\s*$$" | sort > $$ENV_LOCAL_SORTED; \
		\
		echo ""; \
		echo "Keys only in .env:"; \
		comm -23 <(cut -d= -f1 $$ENV_SORTED) <(cut -d= -f1 $$ENV_LOCAL_SORTED) || true; \
		\
		echo ""; \
		echo "Keys only in .env.local:"; \
		comm -13 <(cut -d= -f1 $$ENV_SORTED) <(cut -d= -f1 $$ENV_LOCAL_SORTED) || true; \
		\
		echo ""; \
		echo "Keys with different values:"; \
		awk -F= '\''NR==FNR { env[$$1] = $$2; next } \
			{ if ($$1 in env && env[$$1] != $$2) \
				printf "%s\n.env: %s\n.env.local: %s\n\n", $$1, env[$$1], $$2 }'\'' \
			$$ENV_SORTED $$ENV_LOCAL_SORTED || true; \
		rm -f $$ENV_SORTED $$ENV_LOCAL_SORTED; \
	'
	@echo ""

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
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/postgres.mk),)
	include $(PROJECT_HOME)/resources/make/postgres.mk
else
	$(warning $(PROJECT_HOME)/resources/make/postgres.mk not found — postgres targets not loaded)
endif

ifneq ($(wildcard $(PROJECT_HOME)/resources/make/kafka.mk),)
	include $(PROJECT_HOME)/resources/make/kafka.mk
else
	$(warning $(PROJECT_HOME)/resources/make/kafka.mk not found — kafka targets not loaded)
endif

ifneq ($(wildcard $(PROJECT_HOME)/resources/make/certificates.mk),)
	include $(PROJECT_HOME)/resources/make/certificates.mk
else
	$(warning $(PROJECT_HOME)/resources/make/certificates.mk not found — certificates targets not loaded)
endif

ifneq ($(wildcard $(PROJECT_HOME)/resources/make/minio.mk),)
	include $(PROJECT_HOME)/resources/make/minio.mk
else
	$(warning $(PROJECT_HOME)/resources/make/minio.mk not found — minio targets not loaded)
endif

ifneq ($(wildcard $(PROJECT_HOME)/resources/make/kubernetes.mk),)
	include $(PROJECT_HOME)/resources/make/kubernetes.mk
else
	$(warning $(PROJECT_HOME)/resources/make/kubernetes.mk not found — kubernetes targets not loaded)
endif

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

MODELS := $(shell find resources/postgresql/models/ -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

# ------------------------------------------------------------------
# --- Targets ---
# ------------------------------------------------------------------

.PHONY: info install services-build services-run services-test services-lint services-clean \
        services-docker-build \
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
	@$(MAKE) separator
