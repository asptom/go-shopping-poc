# Sub-Makefile for PostgreSQL installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/postgres.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: postgres-info postgres-initialize postgres-wait \
        postgres-create-dbs postgres-create-schemas

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
postgres-info: ## Show PostgreSQL configuration details
	@$(MAKE) separator
	@echo "PostgreSQL Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $$PROJECT_HOME"
	@echo "Namespace: $$PSQL_NAMESPACE"
	@echo "Models Dir: $$PSQL_MODELS_DIR"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Wait target
# ------------------------------------------------------------------
postgres-wait:
	@echo "Waiting for postgres pod to be Ready..."
	@while true; do \
		status=$$(kubectl -n $$PSQL_NAMESPACE get pods -l app=postgres \
			-o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo ""); \
		if [[ "$$status" == "True" ]]; then break; fi; \
		echo "Waiting for postgresql pod..."; \
		sleep 5; \
	done
	@echo "Postgres pod is Ready."

# ------------------------------------------------------------------
# Create DBs
# ------------------------------------------------------------------
postgres-create-dbs:
	@$(MAKE) separator
	@echo "Creating databases..."
	@cd "$$PSQL_MODELS_DIR"
	@if ! command -v envsubst &>/dev/null; then \
		echo "envsubst not found. Please install."; exit 1; \
	fi
	@for template in $(shell find . -type f -name '*_db.sql' | sort); do \
		outputdb="$${template%_db.sql}.db"; \
		[ -f "$$outputdb" ] && rm "$$outputdb"; \
		echo "Processing database template: $$template"; \
		envsubst < "$$template" > "$$outputdb" || { echo "envsubst failed for $$template"; continue; }; \
		[ -s "$$outputdb" ] || { echo "Output file $$outputdb empty. Skipping."; rm -f "$$outputdb"; continue; }; \
		DB=$$(grep -m1 '^-- DB:' "$$outputdb" | sed 's/^-- DB:[[:space:]]*//'); \
		USER=$$(grep -m1 '^-- USER:' "$$outputdb" | sed 's/^-- USER:[[:space:]]*//'); \
		PGPASSWORD=$$(grep -m1 '^-- PGPASSWORD:' "$$outputdb" | sed 's/^-- PGPASSWORD:[[:space:]]*//'); \
		if [[ -z "$$DB" || -z "$$USER" || -z "$$PGPASSWORD" ]]; then \
			echo "Missing DB, USER, or PGPASSWORD in $$outputdb. Skipping."; rm -f "$$outputdb"; continue; \
		fi; \
		echo "Creating database $$DB..."; \
		kubectl -n $$PSQL_NAMESPACE run psql-client --rm -i --restart='Never' \
			--image=docker.io/postgres:18.0 --env="PGPASSWORD=$$PGPASSWORD" --command -- \
			psql --host postgres -U "$$USER" -d "$$DB" -p 5432 < "$$outputdb" || { echo "Failed $$outputdb"; exit 1; }; \
		rm -f "$$outputdb"; \
	done
	@echo "Databases created."

# ------------------------------------------------------------------
# Create Schemas
# ------------------------------------------------------------------
postgres-create-schemas:
	@$(MAKE) separator
	@echo "Creating schemas..."
	@cd "$$PSQL_MODELS_DIR"
	@for template in $(shell find . -type f -name '*_db.sql' | sort); do \
		sqlfile="$${template%_db.sql}_schema.sql"; \
		[ -f "$$sqlfile" ] || continue; \
		outputsql="$${sqlfile%.sql}_substituted.sql"; \
		[ -f "$$outputsql" ] && rm "$$outputsql"; \
		envsubst < "$$sqlfile" > "$$outputsql" || { echo "envsubst failed for $$sqlfile"; exit 1; }; \
		DB=$$(grep -m1 '^-- DB:' "$$outputsql" | sed 's/^-- DB:[[:space:]]*//'); \
		USER=$$(grep -m1 '^-- USER:' "$$outputsql" | sed 's/^-- USER:[[:space:]]*//'); \
		PGPASSWORD=$$(grep -m1 '^-- PGPASSWORD:' "$$outputsql" | sed 's/^-- PGPASSWORD:[[:space:]]*//'); \
		if [[ -z "$$DB" || -z "$$USER" || -z "$$PGPASSWORD" ]]; then \
			echo "Missing DB, USER, or PGPASSWORD in $$outputsql. Skipping."; rm -f "$$outputsql"; continue; \
		fi; \
		echo "Creating schema for $$DB..."; \
		kubectl -n $$PSQL_NAMESPACE run psql-client --rm -i --restart='Never' \
			--image=docker.io/postgres:18.0 --env="PGPASSWORD=$$PGPASSWORD" --command -- \
			psql --host postgres -U "$$USER" -d "$$DB" -p 5432 < "$$outputsql" || { echo "Failed $$outputsql"; exit 1; }; \
		rm -f "$$outputsql"; \
	done
	@echo "Schemas created."

# ------------------------------------------------------------------
# Initialize (calls the above sequentially, inlined)
# ------------------------------------------------------------------
postgres-initialize: ## Initialize databases
	@$(MAKE) separator
	@echo "Starting PostgreSQL install..."
	@[ -d "$$PSQL_MODELS_DIR" ] || { echo "Models dir missing: $$PSQL_MODELS_DIR"; exit 1; }
	@$(MAKE) postgres-wait
	@$(MAKE) postgres-create-dbs
	@$(MAKE) postgres-create-schemas
	@echo "Postgres initialization complete."
