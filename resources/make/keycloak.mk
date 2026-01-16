# Sub-Makefile for Keycloak installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/keycloak.mk

AUTH_NAMESPACE ?= keycloak
KEYCLOAK_REALM_FILE := $(PROJECT_HOME)/resources/keycloak/pocstore-realm.json

# ------------------------------------------------------------------
# Keycloak database secret creation
# ------------------------------------------------------------------

$(eval $(call help_entry,keycloak-db-secret,Keycloak,Create Keycloak database secret in Kubernetes))
.PHONY: keycloak-db-secret
keycloak-db-secret:
	$(call run,Create Keycloak database secret in Kubernetes,$@, \
	$(call db_secret,keycloak,$(AUTH_NAMESPACE)))

# ------------------------------------------------------------------
# Keycloak database init configmap creation
# ------------------------------------------------------------------

$(eval $(call help_entry,keycloak-db-configmap,Keycloak,Create Keycloak database init configmap in Kubernetes))
.PHONY: keycloak-db-configmap
keycloak-db-configmap:
	$(call run,Create Keycloak database init configmap in Kubernetes,$@, \
	$(call db_configmap_init_sql,keycloak,$(AUTH_NAMESPACE)))

# ------------------------------------------------------------------
# Load Keycloak realm configmap into Kubernetes
# ------------------------------------------------------------------
$(eval $(call help_entry,keycloak-realm-configmap,Keycloak,Load the Keycloak realm configuration into a configmap))
.PHONY: keycloak-realm-configmap
keycloak-realm-configmap:
	$(call run,Load the Keycloak realm configuration into a configmap,$@, \
		set -euo pipefail; \
		kubectl -n $(AUTH_NAMESPACE) create configmap keycloak-realm --from-file=realm.json=$(KEYCLOAK_REALM_FILE) --dry-run=client -o yaml | kubectl apply -f -; \
	)
# ------------------------------------------------------------------
# Install Keycloak Statefulset
# ------------------------------------------------------------------

$(eval $(call help_entry,keycloak-install,Keycloak,Install Keycloak statefulset in Kubernetes))
.PHONY: keycloak-install
keycloak-install: keycloak-db-secret keycloak-db-configmap keycloak-realm-configmap
	$(call run,Install Keycloak statefulset in Kubernetes,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/keycloak/db/; \
		kubectl apply -f deploy/k8s/platform/keycloak/; \
		kubectl rollout status statefulset/keycloak -n $(AUTH_NAMESPACE) --timeout=180s; \
	)

# ------------------------------------------------------------------
# Install Keycloak platform
# ------------------------------------------------------------------

$(eval $(call help_entry,keycloak-platform,Keycloak,Install Keycloak platform in Kubernetes))
.PHONY: keycloak-platform
keycloak-platform: keycloak-db-configmap keycloak-realm-configmap keycloak-install