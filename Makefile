# Default environment
ENV ?= development
ENV_FILE = .env.$(ENV)

# compute project root as the directory that contains this Makefile
PROJECT_HOME := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

# run each recipe body in one shell (avoids multiline if/for problems)
.ONESHELL:
SHELL := /bin/bash

# include kafka makefile using PROJECT_HOME (was using PROJECT_ROOT which is undefined)
include $(PROJECT_HOME)/scripts/Makefile/kafka-topics.mk

SERVICES = customer eventreader
MODELS = $(shell find resources/postgresql/models/ -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

.PHONY: all build sqlc run test lint clean docker-build k8s-deploy-postgresql k8s-deploy-kafka k8s-deploy-app k8s-delete-kafka k8s-delete-postgresql k8s-delete-app run-service build-service

all: clean build docker-build k8s-deploy-postgresql k8s-deploy-kafka k8s-deploy-app

build:
	@echo "Building all services..."
	@for svc in $(SERVICES); do \
	    echo "Building $$svc..."; \
	    GOOS=linux GOARCH=amd64 go build -o bin/$$svc ./cmd/$$svc; \
	done

run:
	@echo "Running all services (in background)..."
	@for svc in $(SERVICES); do \
	    echo "Running $$svc with $(ENV_FILE)..."; \
	    APP_ENV=$(ENV) go run ./cmd/$$svc & \
	done

test:
	@echo "Running tests..."
	go test ./...

lint:
	@echo "Linting..."
	golangci-lint run ./...

clean:
	@echo "Cleaning binaries..."
	rm -rf bin/*

# Example: make run-service SERVICE=eventreader ENV=development
run-service:
	@echo "Running $(SERVICE) with $(ENV_FILE)..."
	APP_ENV=$(ENV) go run ./cmd/$(SERVICE)

# Example: make build-service SERVICE=eventreader
build-service:
	@echo "Building $(SERVICE)..."
	GOOS=linux GOARCH=amd64 go build -o bin/$(SERVICE) ./cmd/$(SERVICE)

docker-build:
	@for svc in $(SERVICES); do \
	    echo "Building Docker image for $$svc..."; \
	    docker build -t localhost:5000/go-shopping-poc/$$svc:1.0 -f cmd/$$svc/Dockerfile . ; \
	    docker push localhost:5000/go-shopping-poc/$$svc:1.0; \
	    echo "Docker image for $$svc built and pushed."; \
	done

k8s-deploy-kafka:
	@echo
	@echo "Deploying Kafka to Kubernetes..."
	@echo "--------------------------------"
	@if [ ! -f deployments/kubernetes/kafka/kafka-deploy.yaml ]; then \
	    echo "--> Deployment file for kafka not found, skipping..."; \
	else \
	    kubectl apply -f deployments/kubernetes/kafka/kafka-deploy.yaml; \
	fi
	@echo "--> kafka deployed to Kubernetes."
	@echo "--------------------------------"

k8s-deploy-postgresql:
	@echo
	@echo "Deploying PostgreSQL to Kubernetes..."
	@echo "--------------------------------"			
	@if [ ! -f deployments/kubernetes/postgresql/postgresql-deploy.yaml ]; then \
	    echo "--> Deployment file for postgresql not found, skipping..."; \
	else \
	    kubectl apply -f deployments/kubernetes/postgresql/postgresql-deploy.yaml; \
	fi
	@echo "--> postgresql deployed to Kubernetes."
	@echo "--------------------------------"

k8s-deploy-app:
	@echo
	@echo "Deploying application to Kubernetes..."
	@echo "--------------------------------------"
	@for svc in $(SERVICES); do \
	    echo "-> Deploying $$svc... as StatefulSets"; \
	    if [ ! -f deployments/kubernetes/$$svc/$$svc-deploy.yaml ]; then \
	        echo "--> Deployment file for $$svc not found, skipping..."; \
	    else \
	        kubectl apply -f deployments/kubernetes/$$svc/$$svc-deploy.yaml; \
	    fi; \
	    echo "--> $$svc deployed to Kubernetes."; \
	    echo "--------------------------------"; \
	done
	@echo "All services deployed to Kubernetes with configurations from deployments/kubernetes."
	@echo

k8s-delete-kafka:
	@echo
	@echo "Deleting Kafka from Kubernetes..."
	@echo "--------------------------------"
	@if [ ! -f deployments/kubernetes/kafka/kafka-deploy.yaml ]; then \
	    echo "--> Deployment file for kafka not found, skipping..."; \
	else \
	    kubectl delete -f deployments/kubernetes/kafka/kafka-deploy.yaml; \
	fi
	@echo "--> kafka deleted from Kubernetes."
	@echo "--------------------------------"

k8s-delete-postgresql:
	@echo
	@echo "Deleting PostgreSQL from Kubernetes..."
	@echo "--------------------------------"			
	@if [ ! -f deployments/kubernetes/postgresql/postgresql-deploy.yaml ]; then \
	    echo "--> Deployment file for postgresql not found, skipping..."; \
	else \
	    kubectl delete -f deployments/kubernetes/postgresql/postgresql-deploy.yaml; \
	fi
	@echo "--> postgresql deleted from Kubernetes."
	@echo "--------------------------------"

k8s-delete-app:
	@echo 
	@echo "Deleting Kubernetes deployments..."
	@echo "--------------------------------"
	@for svc in $(SERVICES); do \
	    echo "--> Deleting $$svc..."; \
	    if [ -f deployments/kubernetes/$$svc/$$svc-deploy.yaml ]; then \
	        kubectl delete -f deployments/kubernetes/$$svc/$$svc-deploy.yaml; \
	    fi; \
	    echo "--> Deleted $$svc from kubernetes."; \
	    echo "--------------------------------"; \
	done
	kubectl delete namespace shopping
	@echo "All services deleted from Kubernetes."
	@echo "--------------------------------"