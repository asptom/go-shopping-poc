# Sub-Makefile for TLS certificate installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/resources/make/certificates.mk

SERVICES_NAMESPACE ?= shopping
AUTH_NAMESPACE ?= keycloak

TLS_CONFIGURATION_DIR := $(PROJECT_HOME)/resources/security/tls_certificates/configuration
TLS_CERTIFICATES_DIR := $(PROJECT_HOME)/resources/security/tls_certificates/certificates

# ------------------------------------------------------------------
# Generate certificates target
# ------------------------------------------------------------------
$(eval $(call help_entry,certificates-generate,Certificates,Generate new TLS certificates for pocstore.local and keycloak.local))
.PHONY: certificates-generate
certificates-generate:
	$(call run,Generate TLS certificates,$@, \
		set -euo pipefail; \
		[ -d "$(TLS_CONFIGURATION_DIR)" ] || { echo "Configuration directory missing: $(TLS_CONFIGURATION_DIR)"; exit 1; }; \
		[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }; \
		cd "$(TLS_CERTIFICATES_DIR)"; \
		rm ./pocstore.*;\
		openssl genrsa -out pocstore.key 4096; \
		openssl req -new -sha256 -out pocstore.csr -key pocstore.key -config ../configuration/pocstore-tls-certificate.conf; \
		openssl req -text -noout -in pocstore.csr; \
		openssl x509 -req -days 3650 -in pocstore.csr -signkey pocstore.key -out pocstore.crt -extensions req_ext -extfile ../configuration/pocstore-tls-certificate.conf; \
		rm ./keycloak.*; \
		openssl genrsa -out keycloak.key 4096; \
		openssl req -new -sha256 -out keycloak.csr -key keycloak.key -config ../configuration/keycloak-tls-certificate.conf; \
		openssl req -text -noout -in keycloak.csr; \
		openssl x509 -req -days 3650 -in keycloak.csr -signkey keycloak.key -out keycloak.crt -extensions req_ext -extfile ../configuration/keycloak-tls-certificate.conf; \
		cd $(PROJECT_HOME); \
	)

# ------------------------------------------------------------------
# Keychain target
# ------------------------------------------------------------------
$(eval $(call help_entry,certificates-keychain,Certificates,Install the generated certificates in the system keychain (macOS)))
.PHONY: certificates-keychain
certificates-keychain:
	$(call run,Generate TLS certificates,$@, \
		set -euo pipefail; \
		[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }; \
		cd "$(TLS_CERTIFICATES_DIR)"; \
		sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain pocstore.crt; \
		sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain keycloak.crt; \
		cd $(PROJECT_HOME); \
	)

# ------------------------------------------------------------------
# Install target
# ------------------------------------------------------------------
$(eval $(call help_entry,certificates-install,Certificates,Install the generated certificates as secrets in the Kubernetes ))
.PHONY: certificates-install
certificates-install:
	$(call run,Install certificates as secrets,$@, \
		set -euo pipefail; \
		\
		[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }; \
		cd "$(TLS_CERTIFICATES_DIR)"; \
		kubectl create secret -n $(SERVICES_NAMESPACE) generic pocstorelocal-tls --from-file=tls.crt=./pocstore.crt --from-file=tls.key=./pocstore.key; \
		kubectl create secret -n $(AUTH_NAMESPACE) generic keycloaklocal-tls --from-file=tls.crt=./keycloak.crt --from-file=tls.key=./keycloak.key; \
		cd $(PROJECT_HOME); \
	)