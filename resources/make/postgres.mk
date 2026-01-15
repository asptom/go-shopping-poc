# Sub-Makefile for PostgreSQL installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/postgres.mk

DB_NAMESPACE ?= database
SERVICES_NAMESPACE ?= shopping
AUTH_NAMESPACE ?= keycloak

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
	@echo
	@echo "Creating PostgreSQL secrets..."
	@NEWPASS=$$(openssl rand -hex 16)
	$(call postgres_secret,postgres-admin-secret,$(DB_NAMESPACE),$$NEWPASS,$(DB_NAMESPACE)) 
	$(call postgres_secret,postgres-admin-bootstrap-secret,$(SERVICES_NAMESPACE),$$NEWPASS,$(DB_NAMESPACE))
	$(call postgres_secret,postgres-admin-bootstrap-secret,$(AUTH_NAMESPACE),$$NEWPASS,$(DB_NAMESPACE))
	@echo

# ------------------------------------------------------------------
# Install Postgres statefulset
# ------------------------------------------------------------------

$(eval $(call help_entry,postgres-install,PostgreSQL,Install PostgreSQL statefulset in Kubernetes))
.PHONY: postgres-install
postgres-install: 
	@echo
	@echo "Installing PostgreSQL statefulset in Kubernetes..."
	@kubectl apply -f deploy/k8s/platform/postgres/
	@kubectl rollout status statefulset/postgres -n $(DB_NAMESPACE) --timeout=180s
	@echo

# ------------------------------------------------------------------
# Install Postgres platform
# ------------------------------------------------------------------
$(eval $(call help_entry,postgres-platform,PostgreSQL,Install PostgreSQL platform in Kubernetes))
.PHONY: postgres-platform
postgres-platform: postgres-secrets postgres-install
	@echo
	@echo "PostgreSQL installation complete."
	@echo
