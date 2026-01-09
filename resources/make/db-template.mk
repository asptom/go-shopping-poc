# Render a template file by replacing placeholders
# $(call render,template,service,namespace)

define render
$(subst __NAMESPACE__,$3,\
$(subst $(SERVICE),$2,\
$(file < $1)))
endef

# --------------------------------------------------------------
# Function to create DB Secret for a service
#
# Usage: @$(call create_db_secret,<service>,<service-namespace>)
# --------------------------------------------------------------
define create_db_secret
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
# Usage: @$(call create_db_init_sql_configmap,<service>,<service-namespace>)
# -----------------------------------------------------------
define create_db_init_sql_configmap
	kubectl -n $(2) create configmap $(1)-db-init-sql \
	  --from-file=resources/postgresql/init \
	  --dry-run=client -o yaml | kubectl apply -f -
endef
