# Sub-Makefile for Kafka installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/kafka.mk

KAFKA_NAMESPACE ?= kafka
KAFKA_BROKER := localhost:9092
KAFKA_RESOURCES_DIR := $(PROJECT_HOME)/resources/kafka
KAFKA_TOPICS_FILE := $(KAFKA_RESOURCES_DIR)/topics.def

# ------------------------------------------------------------------
# Create Kafka Topics
# ------------------------------------------------------------------
$(eval $(call help_entry,kafka-create-topics,Kafka,Create Kafka topics as defined in topics file))
.PHONY: kafka-create-topics
kafka-create-topics: 
	@echo
	@bash -euo pipefail -c '\
		if [ ! -f "$(KAFKA_TOPICS_FILE)" ]; then \
			echo "Missing topics file: $(KAFKA_TOPICS_FILE)"; \
			exit 1; \
		fi; \
		echo "Processing topics from $(KAFKA_TOPICS_FILE)"; \
		POD=$$(kubectl -n "$(KAFKA_NAMESPACE)" get pods -l app=kafka -o jsonpath="{.items[0].metadata.name}"); \
		if [ -z "$${POD}" ]; then \
			echo "No Kafka pod found in namespace: $(KAFKA_NAMESPACE)"; \
			exit 1; \
		fi; \
		echo "Will use kafka pod $$POD to create topics."; \
		while IFS= read -r line; do \
			line="$${line%%#*}"; \
			line="$${line## }"; \
			[ -z "$${line}" ] && continue; \
			IFS=":" read -r name parts repl <<< "$${line}"; \
			parts="$${parts:-3}"; \
			repl="$${repl:-1}"; \
			echo "Creating topic: $$name (partitions=$$parts, replicas=$$repl)"; \
			kubectl -n "$(KAFKA_NAMESPACE)" exec "$${POD}" -- \
				/opt/kafka/bin/kafka-topics.sh \
				--bootstrap-server localhost:9092 \
				--create \
				--topic "$${name}" \
				--partitions "$${parts}" \
				--replication-factor "$${repl}" || true; \
		done < "$(KAFKA_TOPICS_FILE)"; \
	'

# ------------------------------------------------------------------
# Install Kafka Statefulset
# ------------------------------------------------------------------
$(eval $(call help_entry,kafka-install,Kafka,Install Kafka statefulset in Kubernetes))
.PHONY: kafka-install
kafka-install:
	@echo
	@echo "Installing Kafka statefulset in Kubernetes..."
	@kubectl apply -f deploy/k8s/platform/kafka/
	@kubectl rollout status statefulset/kafka -n $(KAFKA_NAMESPACE) --timeout=180s
	@echo

# ------------------------------------------------------------------
# Install Kafka and Create Topics
# ------------------------------------------------------------------
$(eval $(call help_entry,kafka-platform,Kafka,Install Kafka and create topics in Kubernetes))
.PHONY: kafka-platform
kafka-platform: kafka-install kafka-create-topics
	@echo
	@echo "Kafka installation and topic creation complete."
	@echo
