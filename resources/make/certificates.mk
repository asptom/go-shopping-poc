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
	@echo
	@echo "Generating TLS certificates..."
	@[ -d "$(TLS_CONFIGURATION_DIR)" ] || { echo "Configuration directory missing: $(TLS_CONFIGURATION_DIR)"; exit 1; }
	@[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }
	@bash -euo pipefail -c '\
		cd "$(TLS_CERTIFICATES_DIR)"; \
		echo ""; \
		echo "Generating new tls certificates for pocstore.local..."; \
		echo ""; \
		rm ./pocstore.*;\
		openssl genrsa -out pocstore.key 4096; \
		openssl req -new -sha256 -out pocstore.csr -key pocstore.key -config ../configuration/pocstore-tls-certificate.conf; \
		openssl req -text -noout -in pocstore.csr; \
		openssl x509 -req -days 3650 -in pocstore.csr -signkey pocstore.key -out pocstore.crt -extensions req_ext -extfile ../configuration/pocstore-tls-certificate.conf; \
		echo ""; \
		echo "Generating new tls certificates for keycloak.local..."; \
		echo ""; \
		rm ./keycloak.*; \
		openssl genrsa -out keycloak.key 4096; \
		openssl req -new -sha256 -out keycloak.csr -key keycloak.key -config ../configuration/keycloak-tls-certificate.conf; \
		openssl req -text -noout -in keycloak.csr; \
		openssl x509 -req -days 3650 -in keycloak.csr -signkey keycloak.key -out keycloak.crt -extensions req_ext -extfile ../configuration/keycloak-tls-certificate.conf; \
		echo "Certificates generated in $$TLS_CERTIFICATES_DIR"; \
	'

# ------------------------------------------------------------------
# Keychain target
# ------------------------------------------------------------------
$(eval $(call help_entry,certificates-keychain,Certificates,Install the generated certificates in the system keychain (macOS)))
.PHONY: certificates-keychain
certificates-keychain: 
	@echo
	@[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }
	@bash -euo pipefail -c '\
		cd "$(TLS_CERTIFICATES_DIR)"; \
		echo ""; \
		echo "Installing tls certificates for pocstore.local..."; \
		echo ""; \
		sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain pocstore.crt; \
		echo ""; \
		echo "Installing tls certificates for keycloak.local..."; \
		echo ""; \
		sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain keycloak.crt; \
		echo "Certificates installed in $(TLS_CERTIFICATES_DIR)"; \
	'

# ------------------------------------------------------------------
# Install target
# ------------------------------------------------------------------
$(eval $(call help_entry,certificates-install,Certificates,Install the generated certificates as secrets in the Kubernetes ))
.PHONY: certificates-install
certificates-install: 
	@echo
	@[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }
	@bash -euo pipefail -c '\
		cd "$(TLS_CERTIFICATES_DIR)"; \
		echo ""; \
		echo "Installing secret holding tls certificates for pocstore.local..."; \
		echo ""; \
		kubectl create secret -n $(SERVICES_NAMESPACE) generic pocstorelocal-tls --from-file=tls.crt=./pocstore.crt --from-file=tls.key=./pocstore.key; \
		echo ""; \
		echo "Installing secret holding tls certificates for keycloak.local..."; \
		echo ""; \
		kubectl create secret -n $(AUTH_NAMESPACE) generic keycloaklocal-tls --from-file=tls.crt=./keycloak.crt --from-file=tls.key=./keycloak.key; \
		echo "Secrets holding certificates installed"; \
	'

# ------------------------------------------------------------------
# Uninstall target
# ------------------------------------------------------------------
$(eval $(call help_entry,certificates-uninstall,Certificates,Uninstall the generated certificates from the Kubernetes cluster))
.PHONY: certificates-uninstall
certificates-uninstall: 
	@echo
	@[ -d "$(TLS_CERTIFICATES_DIR)" ] || { echo "Deployment directory missing: $(TLS_CERTIFICATES_DIR)"; exit 1; }
	@bash -euo pipefail -c '\
		echo "Removing tls certificates for pocstore.local..."; \
		kubectl -n "$(SERVICES_NAMESPACE)" delete secret pocstorelocal-tls || echo "Secret pocstorelocal-tls not found"; \
		echo ""; \
		echo "Removing tls certificates for keycloak.local..."; \
		kubectl -n "$(AUTH_NAMESPACE)" delete secret keycloaklocal-tls || echo "Secret keycloaklocal-tls not found"; \
	'