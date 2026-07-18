#!/bin/bash
# NexusCore Workspace Bootstrapper script
# Pre-validates configurations, dependencies, and environment files

set -euo pipefail

echo "====================================================================="
echo "       🚀 BOOTSTRAPPING NEXUSCORE ENTERPRISE PLATFORM WORKSPACE      "
echo "====================================================================="

# 1. Check for required command-line tools
echo "Checking workspace system prerequisites..."
for cmd in go docker; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "⚠️  Prerequisite check failed: '$cmd' is not installed."
        echo "   Please make sure '$cmd' is available in your PATH before continuing."
        exit 1
    else
        echo "  ✓ '$cmd' is installed."
    fi
done

# 2. Synchronize module declarations and verify Go Workspace setup
echo "Initializing Go workspaces and synchronizing modules..."
go work sync

# 3. Generating local secrets environment template
if [ ! -f .env ]; then
    echo "Creating .env credentials configuration template..."
    cat <<EOF > .env
# Global cluster development configurations
POSTGRES_USER=postgres
POSTGRES_PASSWORD=enterprise_password_99
POSTGRES_DB=nexuscore_enterprise
REDIS_ADDR=localhost:6379
REDIS_PASS=redis_secure_pass_77
JWT_SECRET=super_cryptographic_secret_hash_key_111
JAEGER_URL=http://localhost:4317
KAFKA_BROKERS=localhost:9092
EOF
    echo "  ✓ Created template .env with base secure development defaults."
fi

echo "====================================================================="
echo "  ✨ BOOTSTRAP COMPLETE! Run 'make up' to start the complete platform."
echo "====================================================================="
