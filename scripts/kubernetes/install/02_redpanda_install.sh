#!/usr/bin/env bash

# This script will be used for the installation of a small redpanda cluster
# and the creation of topics needed for the application

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc"

source "$PROJECT_HOME/scripts/common/load_env.sh"

# Create namespace for our Redpanda cluster
kubectl get namespace | grep -q "^$REDPANDA_CLUSTER_NAMESPACE" || kubectl create namespace $REDPANDA_CLUSTER_NAMESPACE

# Deploy Redpanda

helm install my-redpanda redpanda/redpanda \
  --namespace $REDPANDA_CLUSTER_NAMESPACE \
  --set statefulset.replicas=1 \
  --set storage.persistentVolume.enabled=false \
  --set external.enabled=true \
  --set external.type=NodePort \
  --set external.domain=localhost \
  --set console.enabled=true \
  --set tls.enabled=false

while [[ $(kubectl -n $REDPANDA_CLUSTER_NAMESPACE get pods -l app.kubernetes.io/name=redpanda -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do echo "Waiting for redpanda to be available..." && sleep 5; done

echo
echo "Redpanda pods are available - create redpanda-client and topics"
echo

# Find a running redpanda pod
POD=$(kubectl -n "$REDPANDA_CLUSTER_NAMESPACE" get pods -l app.kubernetes.io/name=redpanda -o jsonpath='{.items[0].metadata.name}')
if [ -z "$POD" ]; then
  echo "No Redpanda pod found in namespace $REDPANDA_CLUSTER_NAMESPACE"
  exit 1
fi

# # Create the topic CustomerEvents (ok if it already exists)
# echo "Creating topic CustomerEvents..."
# kubectl -n "$REDPANDA_CLUSTER_NAMESPACE" exec "$POD" -- rpk topic create CustomerEvents || {
#   echo "Topic creation returned non-zero (it may already exist)"
# }

# # List topics to verify
# echo "Listing topics:"
# kubectl -n "$REDPANDA_CLUSTER_NAMESPACE" exec "$POD" -- rpk topic list
