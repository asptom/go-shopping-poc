#!/usr/bin/env bash
set -euo pipefail

PROJECT_HOME="$(cd "$(dirname "$0")/../../.." && pwd)"

source "$PROJECT_HOME/scripts/common/load_env.sh"

# Create namespace for our shopping poc application (if it does not already exist)
kubectl get namespace | grep -q "^$PROJECT_NAMESPACE" || kubectl create namespace $PROJECT_NAMESPACE

# This script will be used for the installation of the TLS certificates
# and the creation of secrets in the Kubernetes cluster
# for the application  
# It assumes that the certificates are already generated and available in the specified directory 

cd $TLS_CERTIFICATES_DIR || {
    echo "Failed to change directory to $TLS_CERTIFICATES_DIR"
    exit 1
}

# Check if the TLS certificates exist
if [[ ! -f ./pocstore.crt || ! -f ./pocstore.key || ! -f ./keycloak.crt || ! -f ./keycloak.key ]]; then
    echo "TLS certificate files not found in $TLS_CERTIFICATES_DIR. Please ensure the files are present."
    exit 1
fi

kubectl create secret -n $PROJECT_NAMESPACE generic pocstorelocal-tls --from-file=tls.crt=./pocstore.crt --from-file=tls.key=./pocstore.key
if [[ $? -ne 0 ]]; then
    echo "Failed to create secret for pocstorelocal-tls in namespace '$PROJECT_NAMESPACE'."
    exit 1
fi
kubectl create secret -n $PROJECT_NAMESPACE generic keycloaklocal-tls --from-file=tls.crt=./keycloak.crt --from-file=tls.key=./keycloak.key
if [[ $? -ne 0 ]]; then
    echo "Failed to create secret for keycloaklocal-tls in namespace '$PROJECT_NAMESPACE'."
    exit 1
fi

echo "TLS certificate secrets created successfully in namespace '$PROJECT_NAMESPACE'."
