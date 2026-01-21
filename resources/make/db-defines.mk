
# --------------------------------------------------------------
# Function to create DB Secret for a service
#
# Usage: @$(call db_secret,<service>,<service-namespace>)
# --------------------------------------------------------------
define db_secret
	$(eval DB_SECRET_PASS := $(shell openssl rand -hex 16)) \
	$(eval DB_SECRET_RO_PASS := $(shell openssl rand -hex 16)) \
	(set -euo pipefail; \
	kubectl -n $(2) create secret generic $(1)-db-secret \
		--from-literal=DB_NAME=$(1)_db \
		--from-literal=USERNAME=$(1)_user \
		--from-literal=PASSWORD=$(DB_SECRET_PASS) \
		--from-literal=RO_USERNAME=$(1)_rouser \
		--from-literal=RO_PASSWORD=$(DB_SECRET_RO_PASS) \
		--from-literal=DB_URL="postgres://$(1)_user:$(DB_SECRET_PASS)@postgres.postgres.svc.cluster.local:5432/$(1)_db?sslmode=disable" \
		--dry-run=client -o yaml) | kubectl apply -f -
endef

# -----------------------------------------------------------
# Function to create DB configmap for a service
#
# Usage: @$(call db_configmap_init_sql,<service>,<service-namespace>)
# -----------------------------------------------------------
define db_configmap_init_sql
	set -euo pipefail; \
	kubectl -n $(2) create configmap $(1)-db-init-sql \
	  --from-file=resources/postgresql/init \
	  --dry-run=client -o yaml | kubectl apply -f -
endef

# -----------------------------------------------------------
# Function to create DB configmap for a service
#
# Usage: @$(call db_configmap_migrations_sql,<service>,<service-namespace>)
# -----------------------------------------------------------
define db_configmap_migrations_sql
	set -euo pipefail; \
	kubectl -n $(2) create configmap $(1)-db-migrations-sql \
	  --from-file=internal/service/$(1)/migrations \
	  --dry-run=client -o yaml | kubectl apply -f -
endef

# ------------------------------------------------------------------
# Function to initialize DB for a service
#
# Usage: @$(call db_create,<service>,<service-namespace>)
# ------------------------------------------------------------------

define db_create
	set -euo pipefail; \
	kubectl -n $(2) delete job $(1)-db-create --ignore-not-found; \
	kubectl -n $(2) apply -f deploy/k8s/service/$(1)/db/$(1)-db-create.yaml; \
	kubectl wait --for=condition=complete job/$(1)-db-create -n $(2) --timeout=120s
endef

# ------------------------------------------------------------------
# Function to migrate DB for a service
#
# Usage: @$(call db_migrate,<service>,<service-namespace>)
# ------------------------------------------------------------------

define db_migrate
	set -euo pipefail; \
	kubectl -n $(2) delete job $(1)-db-migrate --ignore-not-found; \
	kubectl -n $(2) apply -f deploy/k8s/service/$(1)/db/$(1)-db-migrate.yaml; \
	kubectl wait --for=condition=complete job/$(1)-db-migrate -n $(2) --timeout=120s
endef

# ------------------------------------------------------------------
# Function to retrieve DB credentials for a service
#
# Usage: @$(call db_credentials,<service>,<service-namespace>)
# ------------------------------------------------------------------

define db_credentials
	set -euo pipefail; \
    kubectl -n $(2) get secret $(1)-db-secret -o jsonpath='{.data.USERNAME}' 2>/dev/null | base64 --decode 2>/dev/null || echo "No USERNAME"; \
	echo ""; \
    kubectl -n $(2) get secret $(1)-db-secret -o jsonpath='{.data.PASSWORD}' 2>/dev/null | base64 --decode 2>/dev/null || echo "No PASSWORD"; \
	echo "";
endef