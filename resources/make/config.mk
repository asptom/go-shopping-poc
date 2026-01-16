# Sub-Makefile for config installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/config.mk

# ------------------------------------------------------------------
# Load platform configmap used by services into Kubernetes
# ------------------------------------------------------------------
$(eval $(call help_entry,config-install,Config,Create platform configmap for all services in Kubernetes))
.PHONY: config-install
config-install:
	$(call run,Create platform configmap for all services in Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/config/; \
	)
