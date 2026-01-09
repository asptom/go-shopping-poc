# =========================================================================
# Example Generic DB Bootstrap Recipe (simplified approach)
# =========================================================================
# This recipe avoids complex heredocs and keeps YAML in files,
# which is easier to maintain and test locally.
#
# Key Pattern:
# 1. Create ConfigMap from pre-existing SQL files (kubectl create configmap --from-file)
# 2. Create Secret with generated passwords (kubectl create secret generic)
# 3. Apply a static/templated Job YAML that references those ConfigMap/Secret names
# 4. Wait for Job completion

# SQL file storage pattern:
#   resources/postgresql/init/keycloak-init.sql          (static SQL for keycloak)
#   resources/postgresql/init/customer-init.sql          (static SQL for customer)
#   resources/postgresql/jobs/db-bootstrap-job.yaml.tpl  (generic Job template)

# =========================================================================
# EXAMPLE 1: Create ConfigMap + Secret + Job (three separate kubectl calls)
# =========================================================================

define bootstrap-db-simple
$(eval SERVICE := $1)
$(eval NAMESPACE := $2)
$(eval APP_USER := $(SERVICE)_user)
$(eval APP_PASS := $(shell openssl rand -hex 16))
$(eval RO_USER  := $(SERVICE)_ro_user)
$(eval RO_PASS  := $(shell openssl rand -hex 16))
$(eval DB_NAME  := $(SERVICE)_db)

	@echo "[INFO] Creating ConfigMap for $(SERVICE) in $(NAMESPACE)..."
	kubectl create configmap $(SERVICE)-db-init-sql \
		--from-file=init.sql=resources/postgresql/init/$(SERVICE)-init.sql \
		-n $(NAMESPACE) \
		--dry-run=client -o yaml | kubectl apply -f -

	@echo "[INFO] Creating Secret for $(SERVICE) in $(NAMESPACE)..."
	kubectl create secret generic $(SERVICE)-db-secret \
		-n $(NAMESPACE) \
		--from-literal=db_name=$(DB_NAME) \
		--from-literal=username=$(APP_USER) \
		--from-literal=password=$(APP_PASS) \
		--from-literal=ro_username=$(RO_USER) \
		--from-literal=ro_password=$(RO_PASS) \
		--dry-run=client -o yaml | kubectl apply -f -

	@echo "[INFO] Applying bootstrap Job for $(SERVICE) in $(NAMESPACE)..."
	kubectl apply -f resources/postgresql/jobs/db-bootstrap-job.yaml -n $(NAMESPACE)

	@echo "[INFO] Waiting for Job $(SERVICE)-db-bootstrap to complete..."
	kubectl wait --for=condition=complete job/$(SERVICE)-db-bootstrap \
		-n $(NAMESPACE) --timeout=120s || true

endef

# Usage example in keycloak.mk:
#   keycloak-db-bootstrap:
#   	$(call bootstrap-db-simple,keycloak,keycloak)

# =========================================================================
# EXAMPLE 2: Create ConfigMap + Secret + Templated Job (job YAML in recipe)
# =========================================================================
# If you prefer the Job YAML to be generated but keep it simple:

define bootstrap-db-with-yaml
$(eval SERVICE := $1)
$(eval NAMESPACE := $2)
$(eval APP_USER := $(SERVICE)_user)
$(eval APP_PASS := $(shell openssl rand -hex 16))
$(eval RO_USER  := $(SERVICE)_ro_user)
$(eval RO_PASS  := $(shell openssl rand -hex 16))
$(eval DB_NAME  := $(SERVICE)_db)

	@echo "[INFO] Creating ConfigMap for $(SERVICE) in $(NAMESPACE)..."
	kubectl create configmap $(SERVICE)-db-init-sql \
		--from-file=init.sql=resources/postgresql/init/$(SERVICE)-init.sql \
		-n $(NAMESPACE) \
		--dry-run=client -o yaml | kubectl apply -f -

	@echo "[INFO] Creating Secret for $(SERVICE) in $(NAMESPACE)..."
	kubectl create secret generic $(SERVICE)-db-secret \
		-n $(NAMESPACE) \
		--from-literal=db_name=$(DB_NAME) \
		--from-literal=username=$(APP_USER) \
		--from-literal=password=$(APP_PASS) \
		--from-literal=ro_username=$(RO_USER) \
		--from-literal=ro_password=$(RO_PASS) \
		--dry-run=client -o yaml | kubectl apply -f -

	@echo "[INFO] Applying bootstrap Job for $(SERVICE) in $(NAMESPACE)..."
	kubectl apply -f - <<'EOF'
apiVersion: batch/v1
kind: Job
metadata:
  name: $(SERVICE)-db-bootstrap
  namespace: $(NAMESPACE)
spec:
  ttlSecondsAfterFinished: 300
  backoffLimit: 3
  template:
    spec:
      restartPolicy: OnFailure
      serviceAccountName: db-bootstrap
      containers:
      - name: bootstrap
        image: postgres:18.0
        imagePullPolicy: IfNotPresent
        envFrom:
        - secretRef:
            name: postgres-admin-bootstrap-secret
        - secretRef:
            name: $(SERVICE)-db-secret
        volumeMounts:
        - name: sql-script
          mountPath: /sql
          readOnly: true
        command: ["/bin/bash", "/sql/init.sql"]
      volumes:
      - name: sql-script
        configMap:
          name: $(SERVICE)-db-init-sql
EOF

	@echo "[INFO] Waiting for Job $(SERVICE)-db-bootstrap to complete..."
	kubectl wait --for=condition=complete job/$(SERVICE)-db-bootstrap \
		-n $(NAMESPACE) --timeout=120s || true

endef

# =========================================================================
# HOW IT WORKS:
# =========================================================================
#
# EXAMPLE 1 (ConfigMap + Secret + Job in separate files):
# ─────────────────────────────────────────────────────────
# Pros:
#   • Cleanest: YAML stays in .yaml files, easy to inspect/version-control
#   • Each resource is created via `kubectl create <resource> ... --dry-run=client -o yaml | kubectl apply -f -`
#   •  This is idempotent: re-running won't error if resources already exist
#   • ConfigMap is created from a pre-existing file (resources/postgresql/init/keycloak-init.sql)
#   • SQL stays in a plain .sql file, not embedded in Make (easy to edit, no escaping issues)
#
# Cons:
#   • Requires pre-existing Job YAML file (resources/postgresql/jobs/db-bootstrap-job.yaml)
#   • If Job YAML references hardcoded names, you'd need per-service files or a simple template
#
# File structure needed:
#   resources/postgresql/init/keycloak-init.sql           ← SQL script for keycloak
#   resources/postgresql/init/customer-init.sql           ← SQL script for customer
#   resources/postgresql/jobs/db-bootstrap-job.yaml       ← Generic Job YAML (refs $(SERVICE)-db-init-sql ConfigMap)
#
# ─────────────────────────────────────────────────────────
# EXAMPLE 2 (ConfigMap + Secret + Job in heredoc, but simple):
# ─────────────────────────────────────────────────────────
# Pros:
#   • Single Make recipe: all resources created in one macro invocation
#   • No external Job YAML file needed
#   • Still uses quoted heredoc (<<'EOF') to avoid shell variable expansion
#   • ConfigMap still from file: keeps SQL out of Make
#
# Cons:
#   • Job YAML is in the heredoc (slight maintenance burden), but it's simple/static
#   • Harder to test the Job YAML locally (but still possible: copy the YAML block to a file)
#
# Key difference from your original attempt:
#   • Your original macro tried to embed BOTH the ConfigMap AND the Job in the heredoc,
#     with nested heredocs (<<EOF ... EOF inside <<'EOF' ... 'EOF').
#   • This version keeps ConfigMap creation separate, so the Job heredoc is simpler
#     and the ConfigMap is built from an actual file (like the old keycloak-db-create.yaml.old).
#
# ─────────────────────────────────────────────────────────
# Why the original failed:
# ─────────────────────────────────────────────────────────
# When you had two sequential kubectl apply -f - <<'EOF' blocks in one macro:
#   1. First heredoc: kubectl apply ConfigMap
#   2. Second heredoc: kubectl apply Job
# This works, but when called via $(call ...) in a recipe with @ suppression,
# the output may be suppressed or the second block doesn't execute cleanly.
#
# These examples avoid that by:
#   • Example 1: Using kubectl create configmap/secret (standard commands, not heredocs)
#   • Example 2: Just one heredoc (for the Job), and ConfigMap from file
#
# ─────────────────────────────────────────────────────────
# Usage in your Makefiles:
# ─────────────────────────────────────────────────────────
#
# In keycloak.mk:
#   keycloak-db-bootstrap: ## Initialize Keycloak database
#   	$(call bootstrap-db-simple,keycloak,keycloak)
#
# In customer.mk (or Makefile):
#   customer-db-bootstrap: ## Initialize Customer database
#   	$(call bootstrap-db-simple,customer,shopping)
#
# In a test Make target:
#   test-db-bootstrap:
#   	$(call bootstrap-db-simple,test-service,test-namespace)
#
# ─────────────────────────────────────────────────────────
# Testing locally (before using in Make):
# ─────────────────────────────────────────────────────────
#
# 1. Verify the SQL file exists and is readable:
#    cat resources/postgresql/init/keycloak-init.sql
#
# 2. Test ConfigMap creation:
#    kubectl create configmap keycloak-db-init-sql \
#      --from-file=init.sql=resources/postgresql/init/keycloak-init.sql \
#      -n keycloak --dry-run=client -o yaml | kubectl apply -f -
#    kubectl describe configmap keycloak-db-init-sql -n keycloak
#
# 3. Test Job YAML (paste the Job block into a temp file, then):
#    kubectl apply -f /tmp/job.yaml --dry-run=client
#
# 4. Run the full Make recipe:
#    make keycloak-db-bootstrap
#
# ─────────────────────────────────────────────────────────
# Summary:
# ─────────────────────────────────────────────────────────
# 
# Pick Example 1 if you want maximum clarity and minimal Make logic.
# Pick Example 2 if you prefer a single macro but want to keep SQL in files.
# Both avoid the nested heredoc quoting nightmare of your original approach.
#
