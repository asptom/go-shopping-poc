# Sub-Makefile for PostgreSQL installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/postgres.mk

DB_NAMESPACE ?= database
SERVICES_NAMESPACE ?= shopping
AUTH_NAMESPACE ?= keycloak

POSTGRES_ADMIN_SECRET := postgres-admin-secret
POSTGRES_ADMIN_BOOTSTRAP_SECRET := postgres-admin-bootstrap-secret
POSTGRES_SECRET_USER := POSTGRES_USER
POSTGRES_SECRET_PASSWORD := POSTGRES_PASSWORD

# ------------------------------------------------------------------
# Create Secrets
# ------------------------------------------------------------------

define postgres_secret
	kubectl -n $(2) create secret generic $(1)  \
      --from-literal=POSTGRES_DB=postgres \
      --from-literal=POSTGRES_USER=postgresadmin \
      --from-literal=POSTGRES_PASSWORD=$(3) \
	  --from-literal=POSTGRES_HOST=postgres.$(4).svc.cluster.local \
      --dry-run=client -o yaml | kubectl apply -f -
endef

$(eval $(call help_entry,postgres-secrets,PostgreSQL,Create PostgreSQL secrets in Kubernetes))
.PHONY: postgres-secrets 
postgres-secrets:
	$(call run,Create and verify Kafka topics,$@, \
		set -euo pipefail; \
		NEWPASS=$$(openssl rand -hex 16); \
		$(call postgres_secret,$(POSTGRES_ADMIN_SECRET),$(DB_NAMESPACE),$$NEWPASS,$(DB_NAMESPACE)); \
		$(call postgres_secret,$(POSTGRES_ADMIN_BOOTSTRAP_SECRET),$(SERVICES_NAMESPACE),$$NEWPASS,$(DB_NAMESPACE)); \
		$(call postgres_secret,$(POSTGRES_ADMIN_BOOTSTRAP_SECRET),$(AUTH_NAMESPACE),$$NEWPASS,$(DB_NAMESPACE)); \
	)

# ------------------------------------------------------------------
# Get PostgreSQL credentials
# ------------------------------------------------------------------

$(eval $(call help_entry,postgres-credentials,PostgreSQL,Retrieve PostgreSQL credentials from secret))
.PHONY: postgres-credentials
postgres-credentials:
	$(call run,Retrieve PostgreSQL credentials from secret,$@, \
		set -euo pipefail; \
		kubectl -n $(DB_NAMESPACE) get secret $(POSTGRES_ADMIN_SECRET) -o jsonpath='{.data.$(POSTGRES_SECRET_USER)}' 2>/dev/null | base64 --decode 2>/dev/null || echo "No $(POSTGRES_SECRET_USER)"; \
		echo ""; \
		kubectl -n $(DB_NAMESPACE) get secret $(POSTGRES_ADMIN_SECRET) -o jsonpath='{.data.$(POSTGRES_SECRET_PASSWORD)}' 2>/dev/null| base64 --decode 2>/dev/null || echo "No $(POSTGRES_SECRET_PASSWORD)"; \
		echo ""; \
	)

# ------------------------------------------------------------------
# Install Postgres statefulset
# ------------------------------------------------------------------

$(eval $(call help_entry,postgres-install,PostgreSQL,Install PostgreSQL statefulset in Kubernetes))
.PHONY: postgres-install
postgres-install:
	$(call run,Install PostgreSQL statefulset in Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/postgres/; \
		kubectl rollout status statefulset/postgres -n $(DB_NAMESPACE) --timeout=180s; \
	)

# ------------------------------------------------------------------
# Install Postgres platform
# ------------------------------------------------------------------
$(eval $(call help_entry,postgres-platform,PostgreSQL,Install PostgreSQL platform in Kubernetes))
.PHONY: postgres-platform
postgres-platform: postgres-secrets postgres-install
