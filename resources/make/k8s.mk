# Sub-Makefile for kubernetes installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/k8s.mk

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
.PHONY: k8s-info ## Show Kubernetes configuration details
k8s-info: 
	@$(MAKE) separator
	@echo "Kubernetes Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Load namespaces into Kubernetes
# ------------------------------------------------------------------
.PHONY: k8s-namespaces ## Deploy all required namespaces to Kubernetes
k8s-namespaces: 
	@$(MAKE) separator
	@echo "Creating namespaces in Kubernetes..."
	@echo "------------------------------------"
	@echo
	@kubectl apply -f deploy/k8s/namespaces/
	@echo

# ------------------------------------------------------------------
# Delete namespaces from Kubernetes
# ------------------------------------------------------------------
.PHONY: k8s-delete-namespaces ## Delete all namespaces from Kubernetes
k8s-delete-namespaces:
	@$(MAKE) separator
	@echo "Deleting namespaces from Kubernetes..."
	@echo "-------------------------------------"
	@echo
	@kubectl delete -f deploy/k8s/namespaces/
	@echo
