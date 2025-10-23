#!/usr/bin/env bash
set -euo pipefail

PROJECT_HOME="$(cd "$(dirname "$0")/../../.." && pwd)"

source "$PROJECT_HOME/scripts/common/load_env.sh"

if [ ! -d "$KAFKA_DEPLOYMENT_DIR" ]; then
    echo "Kafka deployment directory does not exist: $KAFKA_DEPLOYMENT_DIR"
    exit 1
fi

# Create namespace for our Kafka cluster
kubectl get namespace | grep -q "^$KAFKA_CLUSTER_NAMESPACE " || kubectl create namespace $KAFKA_CLUSTER_NAMESPACE

# Deploy Kafka
# Ditching helm install from bitnami in favor of applying our own deployment yaml

kubectl apply -f "$KAFKA_DEPLOYMENT_DIR/kafka-deploy.yaml" --namespace "$KAFKA_CLUSTER_NAMESPACE"

while [[ $(kubectl -n $KAFKA_CLUSTER_NAMESPACE get pods -l app=kafka -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do echo "Waiting for kafka to be available..." && sleep 5; done

echo
echo "Kafka is available - run 03_topics_install.sh to create topics"
echo