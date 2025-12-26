#!/bin/bash

# =============================================================================
# Shopping POC - Development Deployment Script
# =============================================================================
# Deploys all services to development environment

set -e  # Exit on error

echo "🚀 Starting Shopping POC development deployment..."
echo ""

# =============================================================================
# Step1: Create Secrets from Templates
# =============================================================================
echo "📋 Step 1: Creating Secrets from templates..."
cd "$(dirname "$0")/base/secrets"

# Copy and check if secrets already exist
if [ ! -f "customer-secret.yaml" ]; then
    echo "  ⚠️  customer-secret.yaml not found. Creating from template..."
    cp customer-secret.yaml.example customer-secret.yaml
    echo "  ✏️  Please edit customer-secret.yaml with actual values"
    echo "  ⚠️  Press Enter when ready to continue, or Ctrl+C to exit..."
    read
fi

if [ ! -f "product-secret.yaml" ]; then
    echo "  ⚠️  product-secret.yaml not found. Creating from template..."
    cp product-secret.yaml.example product-secret.yaml
    echo "  ✏️  Please edit base/secrets/product-secret.yaml with actual values"
    echo "  ⚠️  Press Enter when ready to continue, or Ctrl+C to exit..."
    read
fi

if [ ! -f "postgres-secret.yaml" ]; then
    echo "  ⚠️  postgres-secret.yaml not found. Creating from template..."
    cp postgres-secret.yaml.example postgres-secret.yaml
    echo "  ✏️  Please edit base/secrets/postgres-secret.yaml with actual values"
    echo "  ⚠️  Press Enter when ready to continue, or Ctrl+C to exit..."
    read
fi

if [ ! -f "minio-secret.yaml" ]; then
    echo "  ⚠️  minio-secret.yaml not found. Creating from template..."
    cp minio-secret.yaml.example minio-secret.yaml
    echo "  ✏️  Please edit base/secrets/minio-secret.yaml with actual values"
    echo "  ⚠️  Press Enter when ready to continue, or Ctrl+C to exit..."
    read
fi

if [ ! -f "keycloak-secret.yaml" ]; then
    echo "  ⚠️  keycloak-secret.yaml not found. Creating from template..."
    cp keycloak-secret.yaml.example keycloak-secret.yaml
    echo "  ✏️  Please edit base/secrets/keycloak-secret.yaml with actual values"
    echo "  ⚠️  Press Enter when ready to continue, or Ctrl+C to exit..."
    read
fi

cd ../..
echo "  ✅ Secrets prepared"
echo ""

# =============================================================================
# Step 2: Apply ConfigMaps
# =============================================================================
echo "📋 Step 2: Applying ConfigMaps..."

cd "$(dirname "$0")"

echo "  - Platform ConfigMap..."
kubectl apply -f base/configmaps/platform-configmap.yaml

echo "  - Customer ConfigMap..."
kubectl apply -f base/configmaps/customer-configmap.yaml

echo "  - Product ConfigMap..."
kubectl apply -f base/configmaps/product-configmap.yaml

echo "  - EventReader ConfigMap..."
kubectl apply -f base/configmaps/eventreader-configmap.yaml

echo "  - WebSocket ConfigMap..."
kubectl apply -f base/configmaps/websocket-configmap.yaml

echo "  - EventWriter ConfigMap..."
kubectl apply -f base/configmaps/eventwriter-configmap.yaml

echo "  - PostgreSQL ConfigMap..."
kubectl apply -f base/configmaps/postgresql-configmap.yaml

echo "  - MinIO ConfigMap..."
kubectl apply -f base/configmaps/minio-configmap.yaml

echo "  - Keycloak ConfigMap..."
kubectl apply -f base/configmaps/keycloak-configmap.yaml

echo "  ✅ All ConfigMaps applied"
echo ""

# =============================================================================
# Step 3: Apply Secrets
# =============================================================================
echo "📋 Step 3: Applying Secrets..."

echo "  - Customer Secret..."
kubectl apply -f base/secrets/customer-secret.yaml

echo "  - Product Secret..."
kubectl apply -f base/secrets/product-secret.yaml

echo "  - PostgreSQL Secret..."
kubectl apply -f base/secrets/postgres-secret.yaml

echo "  - MinIO Secret..."
kubectl apply -f base/secrets/minio-secret.yaml

echo "  - Keycloak Secret..."
kubectl apply -f base/secrets/keycloak-secret.yaml

echo "  ✅ All Secrets applied"
echo ""

# =============================================================================
# Step 4: Apply Namespace and Platform Services
# =============================================================================
echo "📋 Step 4: Applying Namespaces and Platform Services..."

echo "  - Namespaces..."
kubectl apply -f base/services/kafka/namespace.yaml

echo "  - PostgreSQL..."
kubectl apply -f postgresql/postgresql-deploy.yaml

echo "  - Kafka..."
kubectl apply -f kafka/kafka-deploy.yaml

echo "  - MinIO..."
kubectl apply -f minio/minio-deploy.yaml

echo "  - Keycloak..."
kubectl apply -f keycloak/keycloak-deploy.yaml

echo "  ✅ Platform services deployed"
echo ""

# =============================================================================
# Step 5: Apply Application Services
# =============================================================================
echo "📋 Step 5: Applying Application Services..."

echo "  - Customer Service..."
kubectl apply -f customer/customer-deploy.yaml

echo "  - Product Service..."
kubectl apply -f product/product-deploy.yaml

echo "  - EventReader Service..."
kubectl apply -f eventreader/eventreader-deploy.yaml

echo "  - WebSocket Service..."
kubectl apply -f websocket/websocket-deployment.yaml
kubectl apply -f websocket/websocket-ingress.yaml
kubectl apply -f websocket/websocket-service.yaml

echo "  - EventWriter Service..."
kubectl apply -f eventwriter/eventwriter-deployment.yaml

echo "  ✅ Application services deployed"
echo ""

# =============================================================================
# Step 6: Verify Deployment
# =============================================================================
echo "📋 Step 6: Verifying Deployment..."

echo "  Checking pods..."
kubectl get pods --all-namespaces --no-headers | grep -E "(customer|product|eventreader|websocket|eventwriter|postgres|kafka|minio|keycloak)" | head -10

echo ""
echo "  Checking ConfigMaps..."
kubectl get configmaps --all-namespaces | grep -E "(platform|customer|product|eventreader|websocket|eventwriter|postgresql|minio|keycloak)"

echo ""
echo "  Checking Secrets..."
kubectl get secrets --all-namespaces | grep -E "(customer|product|postgres|minio|keycloak)"

echo ""
echo "✅ Development deployment complete!"
echo ""
echo "🔍 To view pod logs:"
echo "  kubectl logs -n shopping store-customer-0"
echo ""
echo "🔍 To view pod status:"
echo "  kubectl get pods --all-namespaces"
echo ""
echo "🔄 To restart a service:"
echo "  kubectl rollout restart statefulset/store-customer -n shopping"
echo ""
