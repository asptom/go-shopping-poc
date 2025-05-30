#!/usr/bin/env bash

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc"
NAMESPACE="shopping"

if [ ! -d "$PROJECT_HOME" ]; then
    echo "Project home directory does not exist: $PROJECT_HOME"
    exit 1
fi

cd $PROJECT_HOME/resources/security/tls_certificates/certificates
if [ $? -ne 0 ]; then
    echo "Failed to change directory to $PROJECT_HOME/resources/security/tls_certificates/certificates"
    exit 1
fi

if kubectl get namespace "$NAMESPACE" &>/dev/null; then
    kubectl create secret -n $NAMESPACE generic pocstorelocal-tls --from-file=tls.crt=./pocstore.crt --from-file=tls.key=./pocstore.key
    kubectl create secret -n $NAMESPACE generic keycloaklocal-tls --from-file=tls.crt=./keycloak.crt --from-file=tls.key=./keycloak.key
    echo "TLS certificates created successfully in namespace '$NAMESPACE'."
else
    echo "Namespace '$NAMESPACE' does not exist. Certificate secrets will not be created."
    exit 1
fi