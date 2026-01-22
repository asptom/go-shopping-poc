# Sub-Makefile for Minio installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/minio.mk

MINIO_NAMESPACE ?= minio
SERVICES_NAMESPACE ?= shopping
MINIO_SECRET := minio-secret
MINIO_BOOTSTRAP_SECRET := minio-bootstrap-secret
MINIO_SECRET_USER := MINIO_ROOT_USER
MINIO_SECRET_PASSWORD := MINIO_ROOT_PASSWORD
MINIO_USER := minioadmin

# ------------------------------------------------------------------
# Create Minio Secret and Credentials
# ------------------------------------------------------------------
$(eval $(call help_entry,minio-secret,Minio,Create Minio secret in Kubernetes))
.PHONY: minio-secret
minio-secret:
	$(call run,Create Minio secret in Kubernetes,$@, \
		set -euo pipefail; \
		NEWPASS=$$(openssl rand -hex 16); \
		kubectl -n $(MINIO_NAMESPACE) create secret generic $(MINIO_SECRET) \
			--from-literal=$(MINIO_SECRET_USER)=$(MINIO_USER) \
			--from-literal=$(MINIO_SECRET_PASSWORD)=$$NEWPASS \
			--dry-run=client -o yaml | kubectl apply -f -; \
		kubectl -n $(SERVICES_NAMESPACE) create secret generic $(MINIO_BOOTSTRAP_SECRET) \
			--from-literal=MINIO_ACCESS_KEY=$(MINIO_USER) \
			--from-literal=MINIO_SECRET_KEY=$$NEWPASS \
			--dry-run=client -o yaml | kubectl apply -f -; \
	)

# ------------------------------------------------------------------
# Install Minio Statefulset
# ------------------------------------------------------------------
$(eval $(call help_entry,minio-install,Minio,Install Minio statefulset in Kubernetes))
.PHONY: minio-install
minio-install: minio-secret
	$(call run,Install Minio statefulset in Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/minio/; \
		kubectl rollout status statefulset/minio -n $(MINIO_NAMESPACE) --timeout=180s; \
	)

# ------------------------------------------------------------------
# Get Minio credentials
# ------------------------------------------------------------------

$(eval $(call help_entry,minio-credentials,Minio,Retrieve Minio credentials from secret))
.PHONY: minio-credentials
minio-credentials:
	$(call run,Retrieve Minio credentials from secret,$@, \
		set -euo pipefail; \
		kubectl -n $(MINIO_NAMESPACE) get secret $(MINIO_SECRET) -o jsonpath='{.data.$(MINIO_SECRET_USER)}' 2>/dev/null| base64 --decode 2>/dev/null || echo "No $(MINIO_SECRET_USER)"; \
		echo ""; \
		kubectl -n $(MINIO_NAMESPACE) get secret $(MINIO_SECRET) -o jsonpath='{.data.$(MINIO_SECRET_PASSWORD)}' 2>/dev/null| base64 --decode 2>/dev/null || echo "No $(MINIO_SECRET_USER)"; \
		echo ""; \
	)