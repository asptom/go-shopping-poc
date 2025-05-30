#!/usr/bin/env bash

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc"
if [ ! -d "$PROJECT_HOME" ]; then
    echo "Project home directory does not exist: $PROJECT_HOME"
    exit 1
fi

cd $PROJECT_HOME/resources/security/tls_certificates/configuration
if [ $? -ne 0 ]; then
    echo "Configuration directory does not exist: $PROJECT_HOME/resources/security/tls_certificates/configuration"
    echo "Configuration files are required to generate TLS certificates."
    exit 1
fi

cd $PROJECT_HOME/resources/security/tls_certificates/certificates
if [ $? -ne 0 ]; then
    mkdir -p $PROJECT_HOME/resources/security/tls_certificates/certificates
    echo "Created directory for TLS certificates: $PROJECT_HOME/resources/security/tls_certificates/certificates"
    cd $PROJECT_HOME/resources/security/tls_certificates/certificates
    if [ $? -ne 0 ]; then
        echo "Failed to change directory to $PROJECT_HOME/resources/security/tls_certificates/certificates after creating it"
        exit 1
    fi
fi

echo ""
echo "-----------------------------------------------------------------------"
echo -n "Are you sure that you want to generate new tls certificates? (Y/n): "
read -r generate_new
echo "-----------------------------------------------------------------------"
echo ""

if [[ $generate_new == "y" || $generate_new == "Y" ]]; then
    echo ""
    echo "Generating new tls certificates for pocstore.local..."
    echo ""
    rm ./pocstore.*
    openssl genrsa -out pocstore.key 4096
    openssl req -new -sha256 -out pocstore.csr -key pocstore.key -config ../configuration/pocstore-tls-certificate.conf 
    openssl req -text -noout -in pocstore.csr
    openssl x509 -req -days 3650 -in pocstore.csr -signkey pocstore.key -out pocstore.crt -extensions req_ext -extfile ../configuration/pocstore-tls-certificate.conf
    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain pocstore.crt   
    echo ""
    echo "Generating new tls certificates for keycloak.local..."
    echo ""
    rm ./keycloak.*
    openssl genrsa -out keycloak.key 4096
    openssl req -new -sha256 -out keycloak.csr -key keycloak.key -config ../configuration/keycloak-tls-certificate.conf
    openssl req -text -noout -in keycloak.csr
    openssl x509 -req -days 3650 -in keycloak.csr -signkey keycloak.key -out keycloak.crt -extensions req_ext -extfile ../configuration/keycloak-tls-certificate.conf
    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain keycloak.crt
else
    echo ""
    echo "Will not generate new certificates..."
    echo ""
fi