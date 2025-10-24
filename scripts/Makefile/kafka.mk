# Sub-Makefile for Kafka installation and management
# Include this in your top-level Makefile with:
#   include $(PROJECT_HOME)/scripts/Makefile/kafka.mk

SHELL := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

.PHONY: kafka-info kafka-install kafka-wait \
        kafka-create-topics kafka-uninstall

# ------------------------------------------------------------------
# Info target
# ------------------------------------------------------------------
kafka-info: ## Show Kafka configuration details
	@echo
	@echo "Kafka Configuration:"
	@echo "-------------------------"
	@echo "Project Home: $$PROJECT_HOME"
	@echo "Namespace: $$KAFKA_CLUSTER_NAMESPACE"
	@echo "Broker: $$KAFKA_BROKER"
	@echo "Deployment Dir: $$KAFKA_DEPLOYMENT_DIR"
	@echo "Resources Dir: $$KAFKA_RESOURCES_DIR"
	@echo "Topics File: $$KAFKA_TOPICS_FILE"
	@echo "-------------------------"
	@echo

# ------------------------------------------------------------------
# Wait target
# ------------------------------------------------------------------
kafka-wait:
	@echo "Waiting for kafka pod to be Ready..."
	@while true; do \
		status=$$(kubectl -n $$KAFKA_CLUSTER_NAMESPACE get pods -l app=kafka \
			-o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo ""); \
		if [[ "$$status" == "True" ]]; then break; fi; \
		echo "Waiting for kafka pod..."; \
		sleep 5; \
	done
	@echo "Kafka pod is Ready."

# ------------------------------------------------------------------
# Create Topics
# ------------------------------------------------------------------
kafka-create-topics:
	@bash -euo pipefail -c '\
		if [ ! -f "$$KAFKA_TOPICS_FILE" ]; then \
			echo "Missing topics file: $$KAFKA_TOPICS_FILE"; \
			exit 1; \
		fi; \
		echo "Processing topics from $$KAFKA_TOPICS_FILE"; \
		POD=$$(kubectl -n "$${KAFKA_CLUSTER_NAMESPACE}" get pods -l app=kafka -o jsonpath="{.items[0].metadata.name}"); \
		if [ -z "$${POD}" ]; then \
			echo "No Kafka pod found in namespace: $${KAFKA_CLUSTER_NAMESPACE}"; \
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
			kubectl -n "$${KAFKA_CLUSTER_NAMESPACE}" exec "$${POD}" -- \
				/opt/kafka/bin/kafka-topics.sh \
				--bootstrap-server localhost:9092 \
				--create \
				--topic "$${name}" \
				--partitions "$${parts}" \
				--replication-factor "$${repl}" || true; \
		done < "$$KAFKA_TOPICS_FILE"; \
	'

# ------------------------------------------------------------------
# Install (calls the above sequentially, inlined)
# ------------------------------------------------------------------
kafka-install: ## Deploy Kafka and create topics
	@echo "Starting Kafka install..."
	@[ -d "$$KAFKA_DEPLOYMENT_DIR" ] || { echo "Deployment dir missing: $$KAFKA_DEPLOYMENT_DIR"; exit 1; }
	@[ -d "$$KAFKA_RESOURCES_DIR" ] || { echo "Resources dir missing: $$KAFKA_RESOURCES_DIR"; exit 1; }
	@[ -f "$$KAFKA_TOPICS_FILE" ] || { echo "Topics file missing: $$KAFKA_TOPICS_FILE"; exit 1; }
	@kubectl apply -f "$$KAFKA_DEPLOYMENT_DIR/kafka-deploy.yaml"
	@$(MAKE) kafka-wait
	@$(MAKE) kafka-create-topics
	@echo "Kafka install complete."

# ------------------------------------------------------------------
# Uninstall Kafka
# ------------------------------------------------------------------
kafka-uninstall: ## Remove Kafka deployment
	@echo "Deleting Kafka deployment(namespace: $$KAFKA_CLUSTER_NAMESPACE)..."
	@kubectl -n $$KAFKA_CLUSTER_NAMESPACE delete -f "$$KAFKA_DEPLOYMENT_DIR/kafka-deploy.yaml" || true
	@echo "Uninstall complete."
