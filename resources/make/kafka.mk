# Sub-Makefile for Kafka installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/kafka.mk

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

KAFKA_BROKER := localhost:9092
KAFKA_RESOURCES_DIR := $(PROJECT_HOME)/resources/kafka
KAFKA_TOPICS_FILE := $(KAFKA_RESOURCES_DIR)/topics.def

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
.PHONY: kafka-info ## Show Kafka configuration details
kafka-info:
	@$(MAKE) separator
	@echo "Kafka Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $(PROJECT_HOME)"
	@echo "Namespace: $(KAFKA_NAMESPACE)"
	@echo "Broker: $(KAFKA_BROKER)"
	@echo "Resources Dir: $(KAFKA_RESOURCES_DIR)"
	@echo "Topics File: $(KAFKA_TOPICS_FILE)"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Create Topics
# ------------------------------------------------------------------
.PHONY: kafka-create-topics ## Create Kafka topics as defined in topics file
kafka-create-topics:
	@$(MAKE) separator
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
# Install Kafka platform
# ------------------------------------------------------------------
.PHONY: kafka-install ## Install Kafka in Kubernetes
kafka-install:
	@$(MAKE) separator
	@echo "Installing Kafka in Kubernetes..."
	@kubectl apply -f deploy/k8s/platform/kafka/
	@kubectl rollout status statefulset/kafka -n $(KAFKA_NAMESPACE) --timeout=180s
	@$(MAKE) kafka-create-topics
	@echo
