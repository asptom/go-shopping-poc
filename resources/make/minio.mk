# Sub-Makefile for Minio installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/minio.mk

# ------------------------------------------------------------------
# Install Minio Statefulset
# ------------------------------------------------------------------
$(eval $(call help_entry,minio-install,Minio,Install Minio statefulset in Kubernetes))
.PHONY: minio-install
minio-install:
	$(call run,Install Minio statefulset in Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/minio/; \
		kubectl rollout status statefulset/minio -n $(MINIO_NAMESPACE) --timeout=180s; \
	)