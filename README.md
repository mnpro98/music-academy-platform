# Event-Driven Music Academy Platform

This project implements a modern, scalable, event-driven architecture (EDA) for registering students and processing analytical data in a music academy. 

The system guarantees eventual consistency, fault tolerance, and loose service coupling by leveraging the **Transactional Outbox Pattern**, event streaming, and real-time ETL processing pipelines.

## 🏗️ System Architecture

The end-to-end data flow consists of the following components:

1. **Event Simulator (`event-generator`):** A Go-based microservice that simulates student registration by sending random bursts of mock payloads containing fields like instrument types, program choices, and financial balances at a fully configurable rate.
2. **Ingestion API (`ingestion-api`):** A Go backend that exposes HTTP REST endpoints, validates schemas, and coordinates the transactional storage workflow.
3. **Primary Database (PostgreSQL):** Persistently stores student profiles and serves as the transactional foundation for the **Outbox Pattern**, ensuring data records and outbound events are written atomically.
4. **Event Broker (NATS JetStream):** A distributed messaging platform that reliably queues and persists emitted events for downstream consumption.
5. **Data Processor (Benthos):** A declarative, high-performance streaming engine that pulls events from NATS, executes business filter logic (such as ignoring zero-balance records), and transforms payload structures on the fly.
6. **Analytics Datastore (MongoDB):** An analytical document database that aggregates incoming student ledgers using upsert operations for real-time reporting.

---

## 🛠️ Prerequisites

Before deploying the infrastructure, make sure you have:
- A running Kubernetes cluster locally (**Minikube** or **Kind**).
- `kubectl` configured to communicate with your cluster.
- Docker installed locally for container image compilation.

---

## 🚀 Deployment Instructions

### 1. Prepare the Container Environment
If you are using Minikube, point your terminal to use Minikube's internal Docker daemon so Kubernetes has direct access to your locally built images:
```bash
eval $(minikube docker-env)

### 2. Build the Application Components
Compile and build the Docker images with this custom-made bash script:
```bash
./deploy.sh```

---

## ⚙️ Dynamic Throughput Configuration

The event generator allows you to throttle or accelerate ingestion throughput dynamically at runtime without rebuilding container code, using standard Go duration syntax (250ms, 1s, 5s).

To change the generation interval directly inside the running cluster using vim:

```bash
KUBE_EDITOR="vim" kubectl edit deploy event-generator

Locate the env array block within the pod specification and update the value:

```bash
spec:
  containers:
    - name: event-generator
      env:
        - name: GENERATION_INTERVAL
          value: "500ms"  # <- Edit this string to adjust the send rate

Upon saving and exiting (:wq), Kubernetes automatically performs a rolling restart to spin up a pod with the updated interval configuration.

---

## 🔍 Verification and Monitoring

### Inspect the Analytical Datastore (MongoDB)

To verify that filtered and transformed records successfully exit the Benthos pipeline and land in your analytics layer, exec directly into your MongoDB instance:

```bash
# Execute an interactive shell inside your MongoDB pod
kubectl exec -it <YOUR_MONGODB_POD_NAME> -- mongosh

# Run these commands inside the Mongo shell:
use academy_analytics
db.student_ledgers.countDocuments()
db.student_ledgers.find().sort({ _id: -1 }).limit(5).pretty()
