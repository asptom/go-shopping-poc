#!/bin/bash

# =============================================================================
# Shopping POC - Production Deployment Script
# =============================================================================
# Deploys all services to production environment
# Production uses 2 replicas for critical services

set -e  # Exit on error

echo "🚀 Starting Shopping POC PRODUCTION deployment..."
echo ""

# =============================================================================
# Step 1: Apply ConfigMaps (same as dev)
# =============================================================================
echo "📋 Step 1: Applying ConfigMaps..."

cd "$(dirname "$0")"

kubectl apply -f base/configmaps/platform-configmap.yaml
kubectl apply -f base/configmaps/customer-configmap.yaml
kubectl apply -f base/configmaps/product-configmap.yaml
kubectl apply -f base/configmaps/eventreader-configmap.yaml
kubectl apply -f base/configmaps/websocket-configmap.yaml
kubectl apply -f base/configmaps/eventwriter-configmap.yaml
kubectl apply -f base/configmaps/postgresql-configmap.yaml
kubectl apply -f base/configmaps/minio-configmap.yaml
kubectl apply -f base/configmaps/keycloak-configmap.yaml

echo "  ✅ All ConfigMaps applied"
echo ""

# =============================================================================
# Step 2: Apply Secrets (production values)
# =============================================================================
echo "📋 Step 2: Applying Secrets..."
echo "  ⚠️  Make sure secrets contain production values!"

kubectl apply -f base/secrets/customer-secret.yaml
kubectl apply -f base/secrets/product-secret.yaml
kubectl apply -f base/secrets/postgres-secret.yaml
kubectl apply -f base/secrets/minio-secret.yaml
kubectl apply -f base/secrets/keycloak-secret.yaml

echo "  ✅ All Secrets applied"
echo ""

# =============================================================================
# Step 3: Apply Namespace and Platform Services
# =============================================================================
echo "📋 Step 3: Applying Namespaces and Platform Services..."

kubectl apply -f base/services/kafka/namespace.yaml

kubectl apply -f postgresql/postgresql-deploy.yaml
kubectl apply -f kafka/kafka-deploy.yaml
kubectl apply -f minio/minio-deploy.yaml
kubectl apply -f keycloak/keycloak-deploy.yaml

echo "  ✅ Platform services deployed"
echo ""

# =============================================================================
# Step 4: Apply Application Services (production replicas)
# =============================================================================
echo "📋 Step 4: Applying Application Services with production replicas..."

kubectl apply -f customer/customer-deploy.yaml
kubectl apply -f product/product-deploy.yaml
kubectl apply -f eventreader/eventreader-deploy.yaml
kubectl apply -f websocket/websocket-deployment.yaml
kubectl apply -f websocket/websocket-ingress.yaml
kubectl apply -f websocket/websocket-service.yaml
kubectl apply -f eventwriter/eventwriter-deployment.yaml

echo "  ✅ Application services deployed"
echo ""

# =============================================================================
# Step 2: Apply Secrets (production values)
# =============================================================================
echo "📋 Step 2: Applying Secrets..."
echo "  ⚠️  Make sure secrets contain production values!"

kubectl apply -f base/secrets/customer-secret.yaml
kubectl apply -f base/secrets/product-secret.yaml
kubectl apply -f base/secrets/postgres-secret.yaml
kubectl apply -f base/secrets/minio-secret.yaml
kubectl apply -f base/secrets/keycloak-secret.yaml

echo "  ✅ All Secrets applied"
echo ""

# =============================================================================
# Step 3: Apply Namespace and Platform Services
# =============================================================================
echo "📋 Step 3: Applying Namespaces and Platform Services..."

kubectl apply -f base/services/kafka/namespace.yaml

kubectl apply -f postgresql/postgresql-deploy.yaml
kubectl apply -f kafka/kafka-deploy.yaml
kubectl apply -f minio/minio-deploy.yaml
kubectl apply -f keycloak/keycloak-deploy.yaml

echo "  ✅ Platform services deployed"
echo ""

# =============================================================================
# Step 4: Apply Application Services (production replicas)
# =============================================================================
echo "📋 Step 4: Applying Application Services with production replicas..."

kubectl apply -f customer/customer-deploy.yaml
kubectl apply -f product/product-deploy.yaml
kubectl apply -f eventreader/eventreader-deploy.yaml
kubectl apply -f websocket/websocket-deployment.yaml
kubectl apply -f websocket/websocket-ingress.yaml
kubectl apply -f websocket/websocket-service.yaml
kubectl apply -f eventwriter/eventwriter-deployment.yaml

echo "  ✅ Application services deployed"
echo ""

# =============================================================================
# Step 5: Set Production Replicas
# =============================================================================
echo "📋 Step 5: Setting Production Replicas..."

kubectl scale statefulset/store-customer -n shopping --replicas=2
kubectl scale statefulset/store-product -n shopping --replicas=2

echo "  ✅ Production replicas set (2 for customer, 2 for product)"
echo ""

# =============================================================================
# Step 6: Add Production Labels
# =============================================================================
echo "📋 Step 6: Adding Production Labels..."

kubectl label statefulset/store-customer -n shopping env=production --overwrite
kubectl label statefulset/store-product -n shopping env=production --overwrite
kubectl label statefulset/store-eventreader -n shopping env=production --overwrite
kubectl label deployment/store-websocket -n shopping env=production --overwrite
kubectl label statefulset/store-eventwriter -n shopping env=production --overwrite

echo "  ✅ Production labels applied"
echo ""

# =============================================================================
# Step 7: Verify Deployment
# =============================================================================
echo "📋 Step 7: Verifying Deployment..."

echo "  Checking pods..."
kubectl get pods --all-namespaces | grep -E "(customer|product|eventreader|websocket|eventwriter|postgres|kafka|minio|keycloak)"

echo ""
echo "  Checking ConfigMaps..."
kubectl get configmaps --all-namespaces | grep -E "(platform|customer|product|eventreader|websocket|eventwriter|postgresql|minio|keycloak)"

echo ""
echo "  Checking Secrets..."
kubectl get secrets --all-namespaces | grep -E "(customer|product|postgres|minio|keycloak)"

echo ""
echo "✅ PRODUCTION deployment complete!"
echo ""
echo "🔍 Production Notes:"
echo "  - Replicas: 2 for customer, 2 for product"
echo "  - Labels: env=production on all shopping namespace services"
echo "  - Secrets: Ensure production values in secret files"
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
