#!/usr/bin/env bash
set -euo pipefail

PROJECT_HOME="$(cd "$(dirname "$0")/../../.." && pwd)"

source "$PROJECT_HOME/scripts/common/load_env.sh"
TOPICS_FILE="$PROJECT_HOME/resources/topics/topics.def"
BROKER_TYPE="${BROKER_TYPE:-kafka}" # redpanda | kafka | strimzi


while IFS= read -r line; do
  line="${line%%#*}"       # strip comments
  line="${line## }"        # trim leading
  [ -z "$line" ] && continue
  IFS=':' read -r name parts repl <<< "$line"
  parts="${parts:-3}"
  repl="${repl:-1}"
  echo "Ensuring topic: $name (partitions=$parts replicas=$repl)"
  case "$BROKER_TYPE" in
    redpanda)
      POD=$(kubectl -n "$REDPANDA_CLUSTER_NAMESPACE" get pods -l app.kubernetes.io/name=redpanda -o jsonpath='{.items[0].metadata.name}')
      kubectl -n "$REDPANDA_CLUSTER_NAMESPACE" exec "$POD" -- rpk topic create "$name" --partitions "$parts" --replicas "$repl" || true
      ;;
    kafka)
      POD=$(kubectl -n "$KAFKA_CLUSTER_NAMESPACE" get pods -l app=kafka -o jsonpath='{.items[0].metadata.name}')
      kubectl -n "$KAFKA_CLUSTER_NAMESPACE" exec "$POD" -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --topic "$name" --partitions "$parts" --replication-factor "$repl" || true
      ;;
    strimzi)
      # create KafkaTopic CR; idempotent with kubectl apply
      cat <<EOF | kubectl apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: ${name}
  namespace: ${KAFKA_CLUSTER_NAMESPACE}
  labels:
    strimzi.io/cluster: ${KAFKA_CLUSTER_NAME}
spec:
  partitions: ${parts}
  replicas: ${repl}
EOF
      ;;
    *)
      echo "Unsupported BROKER_TYPE: $BROKER_TYPE" >&2
      exit 1
      ;;
  esac
done < "$TOPICS_FILE"