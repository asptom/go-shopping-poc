# Sub-Makefile for services installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/services.mk

#Service settings
SERVICES := customer eventreader product

SERVICE_HAS_DB_customer := true
SERVICE_HAS_DB_eventreader := false
SERVICE_HAS_DB_product := true

#Kubernetes settings
SERVICES_NAMESPACE ?= shopping

#Docker image settings
IMAGE_REGISTRY ?= localhost:5000
IMAGE_TAG ?= 1.0
IMAGE_PREFIX ?= $(IMAGE_REGISTRY)/go-shopping-poc

# Helper for constructing Docker image names
define image_name
$(IMAGE_PREFIX)/$(1):$(IMAGE_TAG)
endef

# ------------------------------------------------------------------
# Per-service code/deploy targets
# ------------------------------------------------------------------

# Generate per-service code/deploy targets

define SERVICE_TARGETS
$(call assert_arg,service-name,$(1))
$(call dbg,Generating targets for service '$(1)')

$(call help_entry,$(1)-build,Service,Build $(1))
.PHONY: $(1)-build 
$(1)-build:
	$$(call run,Build $(1),$$@,go build -o bin/$(1) ./cmd/$(1))

$(call help_entry,$(1)-test,Service,Test $(1))	
.PHONY: $(1)-test
$(1)-test:	
	$$(call run,Test $(1),$$@,go test ./cmd/$(1)/... ./internal/service/$(1)/...)

$(call help_entry,$(1)-lint,Service,Lint $(1))
.PHONY: $(1)-lint
$(1)-lint:
	$$(call run,Lint $(1),$$@,golangci-lint run ./cmd/$(1)/...)

$(call help_entry,$(1)-clean,Service,Clean $(1))
.PHONY: $(1)-clean
$(1)-clean:
	$$(call run,Clean $(1),$$@,go clean ./cmd/$(1)/...)

$(call help_entry,$(1)-docker,Service,Build and push Docker image for $(1))
.PHONY: $(1)-docker
$(1)-docker:
	$$(call run,Docker image $(1),$$@, \
		set -euo pipefail; \
		docker build \
			-t $$(call image_name,$(1)) \
			-f cmd/$(1)/Dockerfile .; \
		docker push $$(call image_name,$(1)); \
	)

$(call help_entry,$(1)-deploy,Service,Deploy $(1) to Kubernetes)
.PHONY: $(1)-deploy
$(1)-deploy:
	$$(call run,Deploy $(1) to Kubernetes,$$@,\
	set -euo pipefail; \
	kubectl apply -f deploy/k8s/service/$(1)/; \
	kubectl rollout status statefulset/$(1) -n $(SERVICES_NAMESPACE) --timeout=180s; \
	)

$(call help_entry,$(1)-install,Service,Install $(1))
.PHONY: $(1)-install
$(1)-install: $(1)-build $(1)-docker $(1)-deploy
	$$(call run,Install $(1),$$@,echo "$(1) installed")	

$(call help_entry,$(1)-uninstall,Service,Uninstall $(1) from Kubernetes)
.PHONY: $(1)-uninstall
$(1)-uninstall:
	$$(call run,Uninstall $(1) from k8s,$$@,kubectl delete -f deploy/k8s/service/$(1)/ --ignore-not-found)

endef

$(foreach svc,$(strip $(SERVICES)), \
	$(call assert_arg,svc,$(svc)) \
	$(eval $(call SERVICE_TARGETS,$(svc))))

# Aggregate per-service targets

$(eval $(call help_entry,services-build,ServiceAggregate,Build all services))
.PHONY: services-build
services-build: $(addsuffix -build,$(strip $(SERVICES)))

$(eval $(call help_entry,services-test,ServiceAggregate,Test all services))
.PHONY: services-test
services-test: $(addsuffix -test,$(SERVICES))

$(eval $(call help_entry,services-lint,ServiceAggregate,Lint all services))
.PHONY: services-lint
services-lint: $(addsuffix -lint,$(SERVICES))

$(eval $(call help_entry,services-clean,ServiceAggregate,Clean all services))
.PHONY: services-clean
services-clean: $(addsuffix -clean,$(SERVICES))

$(eval $(call help_entry,services-docker,ServiceAggregate,Build and push Docker images for all services))
.PHONY: services-docker
services-docker: $(addsuffix -docker,$(SERVICES))

$(eval $(call help_entry,services-install,ServiceAggregate,Install all services))
.PHONY: services-install
services-install: $(addsuffix -install,$(SERVICES))

$(eval $(call help_entry,services-uninstall,ServiceAggregate,Uninstall all services))
.PHONY: services-uninstall
services-uninstall: $(addsuffix -uninstall,$(SERVICES))	


# ------------------------------------------------------------------
# Per-service database targets
# ------------------------------------------------------------------

# Generate per-service database targets

define SERVICE_HAS_DB_TARGETS
$(call assert_arg,service-name,$(1))
$(call dbg,Generating targets for services with a database '$(1)')

$(call help_entry,$(1)-db-secret,ServiceDB,Create DB secret for $(1))
.PHONY: $(1)-db-secret
$(1)-db-secret:
	$$(call run,DB Secret for $(1),$$@,$(call db_secret,$(1),$(SERVICES_NAMESPACE)))

$(call help_entry,$(1)-db-configmap-init,ServiceDB,Create DB init configmap for $(1))
.PHONY: $(1)-db-configmap-init
$(1)-db-configmap-init:
	$$(call run,DB Init ConfigMap for $(1),$$@,$(call db_configmap_init_sql,$(1),$(SERVICES_NAMESPACE)))

$(call help_entry,$(1)-db-configmap-migrations,ServiceDB,Create DB migrations configmap for $(1))
.PHONY: $(1)-db-configmap-migrations
$(1)-db-configmap-migrations:
	$$(call run,DB Migrations ConfigMap for $(1),$$@,$(call db_configmap_migrations_sql,$(1),$(SERVICES_NAMESPACE)))	

$(call help_entry,$(1)-db-create,ServiceDB,Create DB for $(1))
.PHONY: $(1)-db-create
$(1)-db-create: $(1)-db-secret $(1)-db-configmap-init
	$$(call run,DB Create for $(1),$$@,$(call db_create,$(1),$(SERVICES_NAMESPACE)))

$(call help_entry,$(1)-db-migrate,ServiceDB,Migrate DB for $(1))
.PHONY: $(1)-db-migrate
$(1)-db-migrate: $(1)-db-configmap-migrations
	$$(call run,DB Migrate for $(1),$$@,$(call db_migrate,$(1),$(SERVICES_NAMESPACE)))

$(call help_entry,$(1)-db-credentials,ServiceDB,Get DB credentials for $(1))
.PHONY: $(1)-db-credentials
$(1)-db-credentials:
	$$(call run,Retrieve DB credentials for $(1),$$@,$(call db_credentials,$(1),$(SERVICES_NAMESPACE)))
        
endef

$(foreach svc,$(strip $(SERVICES)), \
	$(call assert_arg,svc,$(svc)) \
	$(if $(call is_true,$(strip $(SERVICE_HAS_DB_$(svc)))), \
		$(eval $(call SERVICE_HAS_DB_TARGETS,$(svc)))) \
)

# Aggregate database targets

$(eval $(call help_entry,services-db-create,ServiceDBAggregate,Create DBs for all services))
.PHONY: services-db-create
services-db-create: \
  $(foreach svc,$(SERVICES), \
    $(if $(call is_true,$(SERVICE_HAS_DB_$(svc))),$(svc)-db-create) \
  )

$(eval $(call help_entry,services-db-migrate,ServiceDBAggregate,Migrate DBs for all services))
.PHONY: services-db-migrate
services-db-migrate: \
  $(foreach svc,$(SERVICES), \
	$(if $(call is_true,$(SERVICE_HAS_DB_$(svc))),$(svc)-db-migrate) \
  )