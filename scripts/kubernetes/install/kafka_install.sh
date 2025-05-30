#!/usr/bin/env bash

# This script will be used for the installation of a small kafka cluster
# and the creation of topics needed for the application

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc/"
KAFKA_CLUSTER_NAME="kafka"
KAFKA_CLUSTER_NAMESPACE="kafka"

# Create namespace for our Kafka cluster
kubectl create namespace $KAFKA_CLUSTER_NAMESPACE

# Deploy Kafka

helm install kafka bitnami/kafka --namespace $KAFKA_CLUSTER_NAMESPACE 

while [[ $(kubectl -n $KAFKA_CLUSTER_NAMESPACE get pods -l app.kubernetes.io/name=kafka -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do echo "Waiting for kafka to be available..." && sleep 10; done

echo
echo "Kafka pod is available - create kafka-client and topics"
echo

kubectl run kafka-client --restart='Never' --image docker.io/bitnami/kafka:3.1.0-debian-10-r8 --namespace kafka --command -- sleep infinity

while [[ $(kubectl -n $KAFKA_CLUSTER_NAMESPACE get pods -l run=kafka-client -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do echo "Waiting for kafka-client to be available..." && sleep 10; done

kubectl exec -it kafka-client --namespace $KAFKA_CLUSTER_NAMESPACE -- bash -c "kafka-topics.sh --create --topic ShoppingCart.events --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"
kubectl exec -it kafka-client --namespace $KAFKA_CLUSTER_NAMESPACE -- bash -c "kafka-topics.sh --create --topic Order.events --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"

echo
echo "List topics"
echo

kubectl exec -it kafka-client --namespace $KAFKA_CLUSTER_NAMESPACE -- bash -c "kafka-topics.sh --list --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"