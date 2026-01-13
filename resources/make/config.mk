# Sub-Makefile for config installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/config.mk

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
.PHONY: config-info ## Show config configuration details
config-info: 
	@$(MAKE) separator
	@echo "Config Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Load configmaps into Kubernetes
# ------------------------------------------------------------------
.PHONY: config-install ## Deploy all required platform configmaps to Kubernetes
config-install: 
	@$(MAKE) separator
	@echo "Creating platform configmaps in Kubernetes..."
	@echo "---------------------------------------------"
	@echo
	@kubectl apply -f deploy/k8s/platform/config/
	@echo
