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
	$(call run,Create and verify Kafka topics,$@, \
		set -euo pipefail; \
		\
		POD=$$(kubectl -n "$(KAFKA_NAMESPACE)" get pods -l app=kafka -o jsonpath="{.items[0].metadata.name}" 2>/dev/null); \
		[ -n "$${POD}" ] || { echo "ERROR: No Kafka pod found"; exit 1; }; \
		\
		[ -f "$(KAFKA_TOPICS_FILE)" ] || { echo "ERROR: Topics file not found"; exit 1; }; \
		\
		echo "Creating topics..."; \
		while IFS= read -r line || [ -n "$$line" ]; do \
			[ -z "$${line%%#*}" ] && continue; \
			line="$${line%%#*}"; \
			line="$${line## }"; \
			[ -z "$${line}" ] && continue; \
			\
			IFS=":" read -r name parts repl <<< "$${line}"; \
			parts="$${parts:-3}"; \
			repl="$${repl:-1}"; \
			\
			kubectl -n "$(KAFKA_NAMESPACE)" exec "$${POD}" -- \
				/opt/kafka/bin/kafka-topics.sh \
				--bootstrap-server localhost:9092 \
				--create \
				--topic "$${name}" \
				--partitions "$${parts}" \
				--replication-factor "$${repl}" || true; \
		done < "$(KAFKA_TOPICS_FILE)"; \
		\
		echo "Verifying topics..."; \
		while IFS= read -r line || [ -n "$$line" ]; do \
			[ -z "$${line%%#*}" ] && continue; \
			line="$${line%%#*}"; \
			line="$${line## }"; \
			[ -z "$${line}" ] && continue; \
			\
			IFS=":" read -r name parts repl <<< "$${line}"; \
			\
			if kubectl -n "$(KAFKA_NAMESPACE)" exec "$${POD}" -- \
				/opt/kafka/bin/kafka-topics.sh \
				--bootstrap-server localhost:9092 \
				--describe \
				--topic "$${name}" >/dev/null 2>&1; then \
				echo "✓ Topic $$name verified"; \
			else \
				echo "✗ Topic $$name verification failed"; \
				exit 1; \
			fi; \
		done < "$(KAFKA_TOPICS_FILE)"; \
	)

# ------------------------------------------------------------------
# Install Kafka Statefulset
# ------------------------------------------------------------------
$(eval $(call help_entry,kafka-install,Kafka,Install Kafka statefulset in Kubernetes))
.PHONY: kafka-install
kafka-install:
	$(call run,Install Kafka statefulset,$@, \
		set -euo pipefail; \
		kubectl apply -f deploy/k8s/platform/kafka/; \
		kubectl rollout status statefulset/kafka -n $(KAFKA_NAMESPACE) --timeout=180s; \
	)
	

# ------------------------------------------------------------------
# Install Kafka and Create Topics
# ------------------------------------------------------------------
$(eval $(call help_entry,kafka-platform,Kafka,Install Kafka and create topics in Kubernetes))
.PHONY: kafka-platform
kafka-platform: kafka-install kafka-create-topics
