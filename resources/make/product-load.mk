# Sub-Makefile for loading products into database
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/product-load.mk

SERVICES_NAMESPACE ?= shopping
PRODUCT_DATABASE_URL := kafka
PRODUCT_CACHE_DIR := $(PROJECT_HOME)/cache
PRODUCT_CSV_FILE := $(PROJECT_HOME)/resources/product-loader/poc-products-short.csv
MINIO_NAMESPACE ?= minio
MINIO_SECRET ?= minio-secret
MINIO_SECRET_USER ?= MINIO_ROOT_USER
MINIO_SECRET_PASSWORD ?= MINIO_ROOT_PASSWORD
MINIO_ENDPOINT ?= http://api.minio.local

# ------------------------------------------------------------------
# Load products into Minio and Postgres database
# ------------------------------------------------------------------
$(eval $(call help_entry,product-load,Product,Load products from CSV into platform))
.PHONY: product-load
product-load:
	$(call run,Load products from CSV into platform,$@, \
		set -euo pipefail; \
		@PRODUCT_DATABASE_URL=$$(shell kubectl -n $(SERVICES_NAMESPACE) get secret product-db-secret -o jsonpath='data.DB_URL_LOCAL' 2>/dev/null | base64 --decode 2>/dev/null || echo "No DB_URL_LOCAL"); \
		@MINIO_ACCESS_KEY=$$(shell kubectl -n $(MINIO_NAMESPACE) get secret $(MINIO_SECRET) -o jsonpath='data.$(MINIO_SECRET_USER)' 2>/dev/null | base64 --decode 2>/dev/null || echo "No $(MINIO_SECRET_USER)"); \
		@MINIO_SECRET_KEY=$$(shell kubectl -n $(MINIO_NAMESPACE) get secret $(MINIO_SECRET) -o jsonpath='data.$(MINIO_SECRET_PASSWORD)' 2>/dev/null| base64 --decode 2>/dev/null || echo "No $(MINIO_SECRET_PASSWORD)"); \
		export DATABASE_URL=$$PRODUCT_DATABASE_URL;\
  		export PRODUCT_CACHE_DIR=$$PRODUCT_CCHE_DIR; \
  		export MINIO_ENDPOINT=$$MINIO_ENDPOINT; \
  		export MINIO_ACCESS_KEY=$$MINIO_ACCESS_KEY; \
  		export MINIO_SECRET_KEY=$$MINIO_SECRET_KEY; \
		go run $(PROJECT_HOME)/cmd/product-loader -csv=$(PRODUCT_CSV_FILE); \
	)