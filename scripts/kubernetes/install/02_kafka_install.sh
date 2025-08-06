#!/usr/bin/env bash

# This script will be used for the installation of a small kafka cluster
# and the creation of topics needed for the application

PROJECT_HOME="/Users/tom/Projects/Go/go-shopping-poc"

source "$PROJECT_HOME/scripts/common/load_env.sh"

# Create namespace for our Kafka cluster
kubectl get namespace | grep -q "^$KAFKA_CLUSTER_NAMESPACE " || kubectl create namespace $KAFKA_CLUSTER_NAMESPACE

# Deploy Kafka

helm install kafka -f $KAFKA_RESOURCES_DIR/kafka-parameters.yaml bitnami/kafka --namespace $KAFKA_CLUSTER_NAMESPACE 

while [[ $(kubectl -n $KAFKA_CLUSTER_NAMESPACE get pods -l app.kubernetes.io/name=kafka -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True True True" ]]; do echo "Waiting for kafka to be available..." && sleep 5; done

echo
echo "Kafka pods are available - create kafka-client and topics"
echo

kubectl run kafka-client --restart='Never' --image docker.io/bitnami/kafka:4.0.0-debian-12-r7 --namespace kafka --command -- sleep infinity

# # Create client.properties file
# # This file will be used to connect to the Kafka cluster when authentication is enabled

# rm -f $KAFKA_RESOURCES_DIR/kafka_client.properties
# cat > $KAFKA_RESOURCES_DIR/kafka_client.properties << EOF
# # security.protocol=SASL_PLAINTEXT
# # sasl.mechanism=PLAIN
# # sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required \
# # username="$KAFKA_USER" \
# # password="$KAFKA_PASSWORD";
# security.protocol=SASL_PLAINTEXT
# sasl.mechanism=SCRAM-SHA-256
# sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required \
#     username="$KAFKA_USER" \
#     password="$KAFKA_PASSWORD";";
# EOF

while [[ $(kubectl -n $KAFKA_CLUSTER_NAMESPACE get pods -l run=kafka-client -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do echo "Waiting for kafka-client to be available..." && sleep 5; done

echo
echo "Delay to ensure kafka-client is ready"
sleep 5
echo

# kubectl cp --namespace $KAFKA_CLUSTER_NAMESPACE ./kafka_client.properties kafka-client:/tmp/client.properties
# kubectl exec --tty -i kafka-client --namespace $KAFKA_CLUSTER_NAMESPACE -- bash -c "kafka-topics.sh --command-config /tmp/client.properties --create --topic EventExampleRead1 --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"

# kubectl exec --tty -i kafka-client --namespace $KAFKA_CLUSTER_NAMESPACE -- bash -c "kafka-topics.sh --create --topic EventExampleRead1 --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"
kubectl exec --tty -i kafka-client --namespace $KAFKA_CLUSTER_NAMESPACE -- bash -c "kafka-topics.sh --create --topic CustomerEvents --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"

echo
echo "List topics"
echo

# kubectl exec --tty -i kafka-client --namespace kafka -- bash -c "kafka-topics.sh --command-config /tmp/client.properties --list --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"

kubectl exec --tty -i kafka-client --namespace kafka -- bash -c "kafka-topics.sh --list --bootstrap-server $KAFKA_CLUSTER_NAME.$KAFKA_CLUSTER_NAMESPACE.svc.cluster.local:9092"
