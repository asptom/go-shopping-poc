# Sub-Makefile for PostgreSQL installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/postgres.mk

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------

.PHONY: postgres-info ## Show PostgreSQL configuration details
postgres-info: 
	@$(MAKE) separator
	@echo "PostgreSQL Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "DB Namespace: $(DB_NAMESPACE)"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Create Secrets
# ------------------------------------------------------------------

.PHONY: postgres-create-secret
postgres-create-secret:
	@kubectl -n $(NAMESPACE) create secret generic $(SECRET_NAME) \
      --from-literal=POSTGRES_DB=postgres \
      --from-literal=POSTGRES_USER=postgresadmin \
      --from-literal=POSTGRES_PASSWORD=$(PASSWORD) \
	  --from-literal=POSTGRES_HOST=postgres.$(DB_NAMESPACE).svc.cluster.local \
      --dry-run=client -o yaml | kubectl apply -f -
	@echo "PostgreSQL Admin secret $(SECRET_NAME) created in namespace $(NAMESPACE)."

.PHONY: postgres-create-secrets  ## Create secrets for Postgres administration use namespaces that need them
postgres-create-secrets: 
	@$(MAKE) separator
	@echo "Creating PostgreSQL secrets..."
	@NEWPASS=$$(openssl rand -hex 16); \
	$(MAKE) postgres-create-secret PASSWORD="$$NEWPASS" NAMESPACE=$(DB_NAMESPACE) SECRET_NAME=postgres-admin-secret; \
	$(MAKE) postgres-create-secret PASSWORD="$$NEWPASS" NAMESPACE=$(SERVICES_NAMESPACE) SECRET_NAME=postgres-admin-bootstrap-secret; \
	$(MAKE) postgres-create-secret PASSWORD="$$NEWPASS" NAMESPACE=$(AUTH_NAMESPACE) SECRET_NAME=postgres-admin-bootstrap-secret
	@echo

# ------------------------------------------------------------------
# Install Postgres platform
# ------------------------------------------------------------------

.PHONY: postgres-install ## Install PostgreSQL platform in Kubernetes
postgres-install: postgres-create-secrets 
	@$(MAKE) separator
	@echo "Installing PostgreSQL platform in Kubernetes..."
	@kubectl apply -f deploy/k8s/platform/postgres/
	@kubectl rollout status statefulset/postgres -n $(DB_NAMESPACE) --timeout=180s
	@echo


# TODO:  Move the migration target to dep-template.mk or a new db-migration.mk


# ------------------------------------------------------------------
# Migrate DBs
# ------------------------------------------------------------------

.PHONY: postgres-migrate-db
postgres-migrate-db:
	@$(MAKE) separator
	@echo "Migrating database..."
	@kubectl delete job $(SERVICE)-db-migrate --ignore-not-found
	@kubectl apply -f deployment/k8s/services/$(SERVICE)-db-migrate.yaml
	@kubectl wait --for=condition=complete job/$(SERVICE)-db-migrate -n $(DB_NAMESPACE) --timeout=120s
	@echo "Database $(SERVICE) migrated."

.PHONY: postgres-migrate-dbs ## Migrate databases for all services
postgres-migrate-dbs:  
	@$(MAKE) separator
	@echo "Migrating databases for all services..."
	@for SERVICE in $(SERVICES_WITH_DB); do \
		$(MAKE) postgres-migrate-db SERVICE=$$SERVICE NAMESPACE=$(DB_NAMESPACE); \
	done
	@echo "All databases migrated."

