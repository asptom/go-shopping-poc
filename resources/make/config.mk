# Sub-Makefile for config installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/config.mk

# ------------------------------------------------------------------
# Load platform configmap used by services into Kubernetes
# ------------------------------------------------------------------
$(eval $(call help_entry,config-install,Config,Create platform configmap for all services in Kubernetes))
.PHONY: config-install
config-install: 
	@echo
	@echo "Creating platform configmaps in Kubernetes..."
	@echo
	@kubectl apply -f deploy/k8s/platform/config/
	@echo
