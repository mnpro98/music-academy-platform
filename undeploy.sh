#!/bin/bash

# Exit immediately if any command fails
set -e

echo "===================================================="
echo "🛑 Starting Music Academy Ecosystem Teardown"
echo "===================================================="

# 1. Verify Minikube is running before trying to run commands
echo "Checking Minikube status..."
if ! minikube status > /dev/null 2>&1; then
    echo "⚠️ Minikube is not running. There is no active cluster to clear."
    exit 0
else
    echo "✓ Minikube is running. Proceeding with purge..."
fi

# 2. Point the terminal's Docker CLI context to Minikube
echo "Configuring environment context..."
eval $(minikube docker-env)

# 3. Delete Kubernetes Manifests in Reverse Sequential Order
echo "----------------------------------------------------"
echo "🗑️ Removing Architecture Resources from Kubernetes..."
echo "----------------------------------------------------"

# Using --ignore-not-found=true ensures the script doesn't crash
# if you already manually deleted a specific resource.

echo "-> Tearing down Mock Event Generator Service..."
kubectl delete -f deployments/k8s/event-generator.yaml --ignore-not-found=true

echo "-> Tearing down Ingestion API & Outbox Worker Service..."
kubectl delete -f deployments/k8s/ingestion-api.yaml --ignore-not-found=true

echo "-> Tearing down Benthos Streaming Pipeline Engine..."
kubectl delete -f deployments/k8s/benthos.yaml --ignore-not-found=true

echo "-> Tearing down NATS JetStream Message Broker..."
kubectl delete -f deployments/k8s/nats.yaml --ignore-not-found=true

echo "-> Tearing down Secondary Database (MongoDB)..."
kubectl delete -f deployments/k8s/secondary-db.yaml --ignore-not-found=true

echo "-> Tearing down Primary Database (PostgreSQL)..."
kubectl delete -f deployments/k8s/primary-db.yaml --ignore-not-found=true

echo "===================================================="
echo "✨ Entire System Successfully Purged!"
echo "===================================================="