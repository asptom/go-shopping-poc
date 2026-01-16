# Sub-Makefile for kubernetes installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/k8s.mk

# ------------------------------------------------------------------
# Load namespaces into Kubernetes
# ------------------------------------------------------------------
$(eval $(call help_entry,k8s-namespaces,Kubernetes,Create all required namespaces in Kubernetes))
.PHONY: k8s-namespaces
k8s-namespaces:
	$(call run,Create namespaces in Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/namespaces/;\
	)

# ------------------------------------------------------------------
# Delete namespaces from Kubernetes
# ------------------------------------------------------------------
$(eval $(call help_entry,k8s-delete-namespaces,Kubernetes,Delete all namespaces from Kubernetes))
.PHONY: k8s-delete-namespaces
k8s-delete-namespaces:
	$(call run,Delete namespaces from Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/namespaces/;\
	)
