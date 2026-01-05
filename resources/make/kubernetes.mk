# Sub-Makefile for kubernetes installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/kubernetes.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: k8s-info k8s-install-namespaces k8s-install-configmaps k8s-install-secrets \
		k8s-install-platform-services k8s-install-domain-services k8s-install \
		k8s-uninstall-domain-services k8s-uninstall-platform-services k8s-uninstall-secrets \
		k8s-uninstall-configmaps k8s-uninstall-namespaces

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
k8s-info: ## Show Kubernetes configuration details
	@$(MAKE) separator
	@echo "Kubernetes Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Keycloak Namespace: $(KEYCLOAK_NAMESPACE)"
	@echo "Keycloak Realm File: $(KEYCLOAK_REALM_FILE)"
	@echo "-------------------------"	
	@echo

# ------------------------------------------------------------------
# Load namespaces into Kubernetes
# ------------------------------------------------------------------
k8s-install-namespaces: ## Deploy all required namespaces to Kubernetes
	@$(MAKE) separator
	@echo "Deploying namespaces to Kubernetes..."
	@echo "-------------------------------------"
	@echo
	@echo "-> Creating namespaces for all services..."
	@for file in deployments/kubernetes/base/namespaces/*.yaml; do \
		echo "--> Deploying namespace: $$file"; \
		kubectl apply -f "$$file"; \
	done
	@echo "All namespaces deployed to Kubernetes."
	@echo

k8s-install-configmaps: ## Deploy all required configmaps to Kubernetes
	@$(MAKE) separator
	@echo "Deploying configmaps to Kubernetes..."
	@echo "-------------------------------------"
	@echo
	@echo "-> Creating base configmaps for all services..."
	@for file in deployments/kubernetes/base/configmaps/*.yaml; do \
		echo "--> Deploying configmap: $$file"; \
		kubectl apply -f "$$file"; \
	done
	@echo "All configmaps deployed to Kubernetes."
	@echo

k8s-install-keycloak-realm-configmap: ## Load the Keycloak realm configuration into a configmap
	@$(MAKE) separator
	@bash -euo pipefail -c '\
		kubectl -n $(KEYCLOAK_NAMESPACE) create configmap keycloak-realm --from-file=realm.json=$(KEYCLOAK_REALM_FILE) --dry-run=client -o yaml | kubectl apply -f -; \
		echo "Keycloak realm configmap created"; \
	'

k8s-install-secrets: ## Deploy all service secrets to Kubernetes
	@$(MAKE) separator
	@echo "Deploying secrets to Kubernetes..."
	@echo "----------------------------------"
	@echo
	@echo "-> Creating base secrets for all services..."
	@for file in deployments/kubernetes/base/secrets/*.yaml; do \
		echo "--> Deploying secret: $$file"; \
		kubectl apply -f "$$file"; \
	done
	@echo "All secrets deployed to Kubernetes."
	@echo

k8s-install-platform-services: ## Deploy all platform services to Kubernetes
	@$(MAKE) separator
	@echo "Deploying platform services to Kubernetes..."
	@echo "--------------------------------------------"
	@echo
	@echo "-> Deploying platform services..."
	@for file in deployments/kubernetes/platform/*.yaml; do \
		echo "--> Deploying platform service: $$file"; \
		kubectl apply -f "$$file"; \
	done
	@echo "All platform services deployed to Kubernetes."
	@echo

k8s-install-domain-services: ## Deploy all domain services to Kubernetes
	@$(MAKE) separator
	@echo "Deploying domain services to Kubernetes..."
	@echo "------------------------------------------"
	@echo
	@echo "-> Deploying domain services..."
	@for file in deployments/kubernetes/services/*.yaml; do \
		echo "--> Deploying domain service: $$file"; \
		kubectl apply -f "$$file"; \
	done
	@echo "All domain services deployed to Kubernetes."
	@echo

k8s-install: k8s-install-namespaces k8s-install-configmaps k8s-install-keycloak-realm-configmap k8s-install-secrets \
		k8s-install-platform-services ## Deploy all required components (except domain services) to Kubernetes
	@$(MAKE) separator
	@echo "Deploying all components to Kubernetes..."
	@echo "-----------------------------------------"
	@echo "All components deployed to Kubernetes."
	@echo

k8s-uninstall-domain-services: ## Remove all domain services from Kubernetes
	@$(MAKE) separator
	@echo "Removing domain services from Kubernetes..."
	@echo "-------------------------------------------"
	@echo
	@echo "-> Removing domain services..."
	@for file in deployments/kubernetes/services/*.yaml; do \
		echo "--> Removing domain service: $$file"; \
		kubectl delete -f "$$file" --ignore-not-found=true; \
	done
	@echo "All domain services removed from Kubernetes."
	@echo

k8s-uninstall-platform-services: ## Remove all platform services from Kubernetes
	@$(MAKE) separator
	@echo "Removing platform services from Kubernetes..."
	@echo "---------------------------------------------"
	@echo
	@echo "-> Removing platform services..."
	@for file in deployments/kubernetes/platform/*.yaml; do \
		echo "--> Removing platform service: $$file"; \
		kubectl delete -f "$$file" --ignore-not-found=true; \
	done
	@echo "All platform services removed from Kubernetes."
	@echo

k8s-uninstall-secrets: ## Remove all service secrets from Kubernetes
	@$(MAKE) separator
	@echo "Removing secrets from Kubernetes..."
	@echo "-----------------------------------"
	@echo
	@echo "-> Removing base secrets for all services..."
	@for file in deployments/kubernetes/base/secrets/*.yaml; do \
		echo "--> Removing secret: $$file"; \
		kubectl delete -f "$$file" --ignore-not-found=true; \
	done
	@echo "All secrets removed from Kubernetes."
	@echo

k8s-uninstall-configmaps: ## Remove all required configmaps from Kubernetes
	@$(MAKE) separator
	@echo "Removing configmaps from Kubernetes..."
	@echo "-------------------------------------"
	@echo
	@echo "-> Removing base configmaps for all services..."
	@for file in deployments/kubernetes/base/configmaps/*.yaml; do \
		echo "--> Removing configmap: $$file"; \
		kubectl delete -f "$$file" --ignore-not-found=true; \
	done
	@echo "All configmaps removed from Kubernetes."
	@echo

k8s-uninstall-namespaces: ## Remove all required namespaces from Kubernetes
	@$(MAKE) separator
	@echo "Removing namespaces from Kubernetes..."
	@echo "-------------------------------------"
	@echo
	@echo "-> Removing namespaces for all services..."
	@for file in deployments/kubernetes/base/namespaces/*.yaml; do \
		echo "--> Removing namespace: $$file"; \
		kubectl delete -f "$$file" --ignore-not-found=true; \
	done
	@echo "All namespaces removed from Kubernetes."
	@echo

k8s-uninstall: k8s-uninstall-domain-services k8s-uninstall-platform-services \
		k8s-uninstall-secrets k8s-uninstall-configmaps k8s-uninstall-namespaces ## Remove all components from Kubernetes
	@$(MAKE) separator
	@echo "All components removed from Kubernetes."
	@echo