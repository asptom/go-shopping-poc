# Sub-Makefile for Minio installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/minio.mk

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

MINIO_NAMESPACE := minio
# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
.PHONY: minio-info ## Show Minio configuration details
minio-info:
	@$(MAKE) separator
	@echo "Minio Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Minio Namespace: $(MINIO_NAMESPACE)"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Install target
# ------------------------------------------------------------------
.PHONY: minio-install ## Install Minio in Kubernetes
minio-install:
	@$(MAKE) separator
	@echo "Installing Minio in Kubernetes..."
	@kubectl apply -f deploy/k8s/platform/minio/
	@echo