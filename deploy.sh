#!/bin/bash

# Exit immediately if any command fails
set -e

echo "===================================================="
echo "🚀 Starting Music Academy Ecosystem Deployment"
echo "===================================================="

# 1. Verify Minikube is running; start it if it isn't
echo "Checking Minikube status..."
if ! minikube status > /dev/null 2>&1; then
    echo "⚠️ Minikube is not running! Starting Minikube now..."
    minikube start
else
    echo "✓ Minikube is running cleanly."
fi

# 2. Point the terminal's Docker CLI context to Minikube
echo "Configuring environment to use Minikube's Docker daemon..."
eval $(minikube docker-env)

# 🛠️ NEW STEP: Synchronize Go dependencies on the host before building
echo "🔄 Syncing Go package modules..."
go mod tidy

# 3. Build Docker Images inside Minikube
echo "----------------------------------------------------"
echo "📦 Building Ingestion API Container Image..."
echo "----------------------------------------------------"
docker build -t ingestion-api:latest -f deployments/docker/IngestionAPI.dockerfile .

echo "----------------------------------------------------"
echo "📦 Building Event Generator Container Image..."
echo "----------------------------------------------------"
docker build -t event-generator:latest -f deployments/docker/EventGenerator.dockerfile .

# 4. Apply Kubernetes Manifests in Sequential Order
echo "----------------------------------------------------"
echo "☸️ Deploying Architecture Resources to Kubernetes..."
echo "----------------------------------------------------"

echo "-> Rolling out Primary Database (PostgreSQL)..."
kubectl apply -f deployments/k8s/primary-db.yaml

echo "-> Rolling out Secondary Database (MongoDB)..."
kubectl apply -f deployments/k8s/secondary-db.yaml

echo "-> Rolling out NATS JetStream Message Broker..."
kubectl apply -f deployments/k8s/nats.yaml

echo "-> Rolling out Benthos Streaming Pipeline Engine..."
kubectl apply -f deployments/k8s/benthos.yaml

echo "-> Rolling out Ingestion API & Outbox Worker Service..."
kubectl apply -f deployments/k8s/ingestion-api.yaml

echo "-> Rolling out Mock Event Generator Service..."
kubectl apply -f deployments/k8s/event-generator.yaml

# Force rolling updates to flush out cached images and ConfigMaps
echo "----------------------------------------------------"
echo "🔄 Executing live cluster rollout restarts..."
echo "----------------------------------------------------"
kubectl rollout restart deployment/ingestion-api
kubectl rollout status deployment/ingestion-api

kubectl rollout restart deployment/benthos-pipeline
kubectl rollout status deployment/benthos-pipeline

echo "===================================================="
echo "✅ Entire System Successfully Deployed to Minikube!"
echo "===================================================="
echo "Stream logs live from your pipeline using this command:"
echo "  kubectl logs -f deployment/benthos-pipeline"
echo "===================================================="