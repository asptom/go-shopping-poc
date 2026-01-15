
# --------------------------------------------------------------
# Function to create DB Secret for a service
#
# Usage: @$(call db_secret,<service>,<service-namespace>)
# --------------------------------------------------------------
define db_secret
	NEWPASS=$$(openssl rand -hex 16); \
	NEWROPASS=$$(openssl rand -hex 16); \
	kubectl -n $(2) create secret generic $(1)-db-secret \
		--from-literal=db_name=$(1)_db \
		--from-literal=username=$(1)_user \
		--from-literal=password=$$NEWPASS \
		--from-literal=ro_username=$(1)_rouser \
		--from-literal=ro_password=$$NEWROPASS \
		--from-literal=db_URL="postgres://$(1)_user:$$NEWPASS@postgres.postgres.svc.cluster.local:5432/$(1)_db?sslmode=disable" \
		--dry-run=client -o yaml | kubectl apply -f -
endef

# -----------------------------------------------------------
# Function to create DB configmap for a service
#
# Usage: @$(call db_configmap_init_sql,<service>,<service-namespace>)
# -----------------------------------------------------------
define db_configmap_init_sql
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
	kubectl -n $(2) delete job $(1)-db-create --ignore-not-found
	kubectl -n $(2) apply -f deploy/k8s/service/$(1)/db/$(1)-db-create.yaml
	kubectl wait --for=condition=complete job/$(1)-db-create -n $(2) --timeout=120s
endef

# ------------------------------------------------------------------
# Function to migrate DB for a service
#
# Usage: @$(call db_migrate,<service>,<service-namespace>)
# ------------------------------------------------------------------

define db_migrate
	kubectl -n $(2) delete job $(1)-db-migrate --ignore-not-found
	kubectl -n $(2) apply -f deploy/k8s/service/$(1)/db/$(1)-db-migrate.yaml
	kubectl wait --for=condition=complete job/$(1)-db-migrate -n $(2) --timeout=120s
endef


