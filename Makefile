# ------------------------------------------------------------------
# Makefile for Go Shopping POC Application
# ------------------------------------------------------------------

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

PROJECT_HOME := /Users/tom/Projects/Go/go-shopping-poc

DB_NAMESPACE := postgres
SERVICES_NAMESPACE := shopping
AUTH_NAMESPACE := keycloak
KAFKA_NAMESPACE := kafka
MINIO_NAMESPACE := minio

# ------------------------------------------------------------------
# --- Include sub-makefiles (use real paths under PROJECT_HOME) ---
# ------------------------------------------------------------------

include $(PROJECT_HOME)/resources/make/make-doctor.mk
include $(PROJECT_HOME)/resources/make/make-debug.mk
include $(PROJECT_HOME)/resources/make/make-utils.mk
include $(PROJECT_HOME)/resources/make/make-help.mk
include $(PROJECT_HOME)/resources/make/make-output.mk
include $(PROJECT_HOME)/resources/make/make-graph.mk

include $(PROJECT_HOME)/resources/make/db-defines.mk
include $(PROJECT_HOME)/resources/make/postgres.mk
include $(PROJECT_HOME)/resources/make/kafka.mk
include $(PROJECT_HOME)/resources/make/certificates.mk
include $(PROJECT_HOME)/resources/make/minio.mk
include $(PROJECT_HOME)/resources/make/k8s.mk
include $(PROJECT_HOME)/resources/make/keycloak.mk
include $(PROJECT_HOME)/resources/make/services.mk
include $(PROJECT_HOME)/resources/make/config.mk
include $(PROJECT_HOME)/resources/make/product_loader.mk

# ------------------------------------------------------------------
# --- Targets ---
# ------------------------------------------------------------------

$(eval $(call help_entry,platform,PRIMARY,Install all platform services (Keycloak, Postgres, Minio, Kafka, Certificates)))
.PHONY: platform 
platform: k8s-namespaces certificates-install postgres-platform keycloak-platform kafka-platform minio-platform config-install

$(eval $(call help_entry,services,PRIMARY,Install all application services))
.PHONY: services
services: services-db-create services-db-migrate services-install

$(eval $(call help_entry,install,PRIMARY,Install all platform services and all application services))
.PHONY: install
install: platform services

$(eval $(call help_entry,uninstall,PRIMARY,Uninstall all services and all supporting components))
.PHONY: uninstall
uninstall: k8s-delete-namespaces

.PHONY: help
help:
	@echo
	@echo "**************************************************************"
	@echo
	@echo "📘 Go Shopping POC — Make Targets"
	@echo "=================================="
	@echo
	@echo "install			Full setup: clean, build, docker-build, create namespaces, install postgres, kafka, certificates, keycloak, minio, and services"
	@echo "uninstall		Uninstall all services, and all supporting components"
	@echo ""
	@echo "Available targets:"
	$(foreach cat,$(HELP_CATEGORIES), \
		echo ""; \
		echo "[$(cat)]"; \
		$(foreach t,$(HELP_TARGETS), \
			$(if $(filter $(cat),$(HELP_CAT_$(subst -,_,$(t)))), \
				printf "  %-32s %s\n" "$(t)" "$(HELP_DESC_$(subst -,_,$(t)))"; \
			) \
		) \
	)
	@echo
	@echo "Usage: make <target>"
	@echo "Example: make postgres-install"
	@echo
	@echo "Usage (with debug): make DBG=1 <target>"
	@echo "Example: make DBG=1 postgres-install"
	@echo
	@echo "**************************************************************"
	@echo
# ------------------------------------------------------------------
# End of Makefile
# ------------------------------------------------------------------