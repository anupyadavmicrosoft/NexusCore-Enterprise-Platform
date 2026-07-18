#!/usr/bin/env bash

# NexusCore Local Kubernetes Environment Bootstrap Script
# Supports Kind (preferred) and Minikube on Windows (WSL2), Linux, and macOS.

set -euo pipefail

# Style definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0;0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 1. Dependency Verification
log_info "Verifying CLI tools..."
for tool in docker kubectl helm; do
    if ! command -v "$tool" &> /dev/null; then
        log_error "$tool is required but not installed. Exiting."
        exit 1
    fi
done
log_success "Basic tools verified."

CLUSTER_PROVIDER="kind"
if ! command -v kind &> /dev/null; then
    if command -v minikube &> /dev/null; then
        CLUSTER_PROVIDER="minikube"
        log_warn "kind not found. Falling back to Minikube."
    else
        log_error "Neither kind nor minikube was found. Please install one of them."
        exit 1
    fi
fi

CLUSTER_NAME="nexuscore-dev"

# 2. Cluster Creation
if [ "$CLUSTER_PROVIDER" = "kind" ]; then
    log_info "Checking if Kind cluster '${CLUSTER_NAME}' already exists..."
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warn "Kind cluster '${CLUSTER_NAME}' already exists. Skipping creation."
    else
        log_info "Creating Kind cluster '${CLUSTER_NAME}'..."
        # Configuration file for Kind with port mappings for the gateway ingress
        cat <<EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: 8080
    listenAddress: "127.0.0.1"
    protocol: TCP
EOF
        kind create cluster --name "$CLUSTER_NAME" --config /tmp/kind-config.yaml
        rm -f /tmp/kind-config.yaml
    fi
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
elif [ "$CLUSTER_PROVIDER" = "minikube" ]; then
    log_info "Checking if Minikube profile '${CLUSTER_NAME}' is active..."
    if minikube profile list | grep -q "${CLUSTER_NAME}"; then
        log_warn "Minikube profile '${CLUSTER_NAME}' already exists. Starting it..."
        minikube start -p "$CLUSTER_NAME"
    else
        log_info "Starting Minikube cluster with profile '${CLUSTER_NAME}'..."
        minikube start -p "$CLUSTER_NAME" --driver=docker
    fi
fi

# 3. Namespace Creation & Core Secret Management
log_info "Setting up namespaces & secrets..."
kubectl create namespace nexuscore --dry-run=client -o yaml | kubectl apply -f -

# Create development database secret
kubectl create secret generic nexuscore-secrets \
    --namespace nexuscore \
    --from-literal=db-password="enterprise_password_99" \
    --from-literal=redis-password="redis_secure_pass_77" \
    --from-literal=jwt-secret="super_cryptographic_secret_hash_key_111" \
    --dry-run=client -o yaml | kubectl apply -f -

# 4. Dependency Deployment via Helm
log_info "Installing Redis & PostgreSQL dependency sub-charts..."
helm repo add bitnami https://charts.bitnami.com/bitnami || true
helm repo update

# Install bitnami postgresql for development
log_info "Deploying PostgreSQL..."
helm upgrade --install postgresql bitnami/postgresql \
    --namespace nexuscore \
    --set auth.database=nexuscore_enterprise \
    --set auth.username=postgres \
    --set auth.password=enterprise_password_99 \
    --set primary.persistence.enabled=false \
    --wait

# Install bitnami redis for development
log_info "Deploying Redis Cache..."
helm upgrade --install redis bitnami/redis \
    --namespace nexuscore \
    --set auth.password=redis_secure_pass_77 \
    --set master.persistence.enabled=false \
    --wait

log_success "Database and cache clusters deployed to Kubernetes successfully!"

# 5. Local Docker Image Compilation and Loading
log_info "Compiling and loading local microservice images..."
SERVICES=("api-gateway" "auth-service" "compute-engine")

for service in "${SERVICES[@]}"; do
    log_info "Building image for: $service..."
    docker build -t "nexuscore/${service}:latest" "./$service"
    
    if [ "$CLUSTER_PROVIDER" = "kind" ]; then
        log_info "Loading image 'nexuscore/${service}:latest' into Kind..."
        kind load docker-image "nexuscore/${service}:latest" --name "$CLUSTER_NAME"
    elif [ "$CLUSTER_PROVIDER" = "minikube" ]; then
        log_info "Loading image 'nexuscore/${service}:latest' into Minikube..."
        minikube image load "nexuscore/${service}:latest" -p "$CLUSTER_NAME"
    fi
done

log_success "All images loaded into cluster."
log_success "NexusCore Local Kubernetes cluster environment is ready for deployment!"
echo -e "${YELLOW}Deploy your microservices using standard kubectl manifests or Helm charts in 'charts/' namespace 'nexuscore'.${NC}"
