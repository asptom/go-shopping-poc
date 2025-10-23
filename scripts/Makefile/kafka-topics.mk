# scripts/Makefile/kafka-topics.mk
# ------------------------------------------------------------------
# Kafka Topic Management Makefile
# ------------------------------------------------------------------
# Usage:
#   make -f scripts/kafka/Makefile kafka-topics
#
# Include this Makefile in your main Makefile with:
#   include scripts/Makefile/kafka-topics.mk
#
# Requires:
#   - A .env file at the project root defining:
#       KAFKA_CLUSTER_NAMESPACE=your-namespace
#       KAFKA_CLUSTER_NAME=your-cluster-name
#   - A topics file at: resources/kafka/topics.def
# ------------------------------------------------------------------

SHELL := /usr/bin/env bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c

# ------------------------------------------------------------------
# Project setup
# ------------------------------------------------------------------
#PROJECT_HOME := $(shell cd $(dir $(lastword $(MAKEFILE_LIST)))/../../.. && pwd)
TOPICS_FILE  := $(PROJECT_HOME)/resources/kafka/topics.def
ENV_FILE     := $(PROJECT_HOME)/.env.development

# Load .env file if it exists
ifneq ("$(wildcard $(ENV_FILE))","")
  include $(ENV_FILE)
  export $(shell sed 's/=.*//' $(ENV_FILE))
else
  $(warning ⚠️  No .env file found at $(ENV_FILE))
endif

# ------------------------------------------------------------------
# Main target
# ------------------------------------------------------------------
.PHONY: kafka-topics
kafka-topics:
	@echo "🔧 Loading environment from $(ENV_FILE)"
	@echo "📦 Project home: $(PROJECT_HOME)"
	@if [ ! -f "$(TOPICS_FILE)" ]; then \
		echo "❌ Missing topics file: $(TOPICS_FILE)"; \
		exit 1; \
	fi
	@echo "Processing topics from $(TOPICS_FILE)"

	@POD=$$(kubectl -n "$${KAFKA_CLUSTER_NAMESPACE}" get pods -l app=kafka -o jsonpath='{.items[0].metadata.name}'); \
	if [ -z "$${POD}" ]; then \
		echo "❌ No Kafka pod found in namespace: $${KAFKA_CLUSTER_NAMESPACE}"; \
		exit 1; \
	fi; \
	while IFS= read -r line; do \
		line="$${line%%#*}"; \
		line="$${line## }"; \
		[ -z "$${line}" ] && continue; \
		IFS=':' read -r name parts repl <<< "$${line}"; \
		parts="$${parts:-3}"; \
		repl="$${repl:-1}"; \
		echo "➡️  Ensuring topic: $${name} (partitions=$${parts}, replicas=$${repl})"; \
		kubectl -n "$${KAFKA_CLUSTER_NAMESPACE}" exec "$${POD}" -- \
			/opt/kafka/bin/kafka-topics.sh \
			--bootstrap-server localhost:9092 \
			--create \
			--topic "$${name}" \
			--partitions "$${parts}" \
			--replication-factor "$${repl}" || true; \
	done < "$(TOPICS_FILE)"
