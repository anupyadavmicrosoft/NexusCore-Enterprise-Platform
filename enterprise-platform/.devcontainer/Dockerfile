FROM mcr.microsoft.com/devcontainers/go:1-1.22-bookworm

# Install extra system tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    redis-tools \
    netcat-openbsd \
    curl \
    git \
    make \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Install Kubectl (for local K8s development)
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

# Install Kind (Kubernetes in Docker)
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64 \
    && chmod +x ./kind \
    && mv ./kind /usr/local/bin/kind

# Install Helm
RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci-lint/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.57.2

# Set standard Go development environment variables
ENV CGO_ENABLED=0
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH
