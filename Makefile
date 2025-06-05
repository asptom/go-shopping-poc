# Default environment
ENV ?= development
ENV_FILE = .env.$(ENV)
PROJECT_ROOT = pwd

# Find all service directories under cmd/
#SERVICES := $(shell find cmd -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)
#SERVICES = websocket
SERVICES = eventwriter eventreader

.PHONY: all build run test lint clean docker-build kubernetes-deploy kubernetes-delete run-service build-service

all: build

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

# Example: make run-service SERVICE=eventexample
run-service:
	@echo "Running $(SERVICE) with $(ENV_FILE)..."
	APP_ENV=$(ENV) go run ./cmd/$(SERVICE)

# Example: make build-service SERVICE=eventexample
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

kubernetes-deploy:
	@echo "Deploying to Kubernetes..."
	@for svc in $(SERVICES); do \
	    echo "Deploying $$svc..."; \
	    kubectl apply -f deployments/kubernetes/$$svc/$$svc-deployment.yaml; \
		kubectl apply -f deployments/kubernetes/$$svc/$$svc-service.yaml; \
		kubectl apply -f deployments/kubernetes/$$svc/$$svc-ingress.yaml; \
	done
	@echo "All services deployed to Kubernetes."

kubernetes-delete:
	@echo "Deleting Kubernetes deployments..."
	@for svc in $(SERVICES); do \
	    echo "Deleting $$svc..."; \
	    kubectl delete -f deployments/kubernetes/$$svc/$$svc-deployment.yaml; \
		kubectl delete -f deployments/kubernetes/$$svc/$$svc-service.yaml; \
		kubectl delete -f deployments/kubernetes/$$svc/$$svc-ingress.yaml; \
	done
	@echo "All services deleted from Kubernetes."