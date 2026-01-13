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

# Services Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/services.mk),)
	include $(PROJECT_HOME)/resources/make/services.mk
else
	$(warning $(PROJECT_HOME)/resources/make/services.mk not found — services targets not loaded)
endif

# Config Makefile
ifneq ($(wildcard $(PROJECT_HOME)/resources/make/config.mk),)
	include $(PROJECT_HOME)/resources/make/config.mk
else
	$(warning $(PROJECT_HOME)/resources/make/config.mk not found — config targets not loaded)
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
	@echo "----------------------------------------"

.PHONY: separator
separator:
	@echo
	@echo "**************************************************************"
	@echo

.PHONY: platform ## Install all platform services (Keycloak, Postgres, Minio, Kafka, Certificates)
platform: k8s-namespaces certificates-install postgres-install keycloak-install kafka-install minio-install config-install

.PHONY: services ## Install all services
services: services-complete-install

.PHONY: install ## Full setup: install all platform services and all application services
install: platform services

.PHONY: uninstall ## Uninstall all services and all supporting components
uninstall: k8s-uninstall

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
