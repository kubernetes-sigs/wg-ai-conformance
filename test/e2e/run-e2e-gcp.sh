#!/usr/bin/env bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ARTIFACTS="${ARTIFACTS:-${REPO_ROOT}/_artifacts}"
mkdir -p "${ARTIFACTS}"

# Configuration
GCE_ZONE="${GCE_ZONE:-us-central1-b}"
GCE_MACHINE_TYPE="${GCE_MACHINE_TYPE:-g2-standard-4}"
GCE_ACCELERATOR="${GCE_ACCELERATOR:-type=nvidia-l4,count=1}"
GCE_IMAGE_FAMILY="${GCE_IMAGE_FAMILY:-common-cu129-ubuntu-2404-nvidia-580}"
GCE_IMAGE_PROJECT="${GCE_IMAGE_PROJECT:-deeplearning-platform-release}"
K8S_VERSION="${K8S_VERSION:-v1.35.0}"
GPU_OPERATOR_VERSION="${GPU_OPERATOR_VERSION:-v26.3.1}"
BUILD_ID="${BUILD_ID:-$(date +%s)}"
VM_NAME="ai-conformance-e2e-${BUILD_ID}"

BOSKOS_HOST="${BOSKOS_HOST:-http://boskos.test-pods.svc.cluster.local}"
BOSKOS_RESOURCE_TYPE="${BOSKOS_RESOURCE_TYPE:-gpu-project}"
JOB_NAME="${JOB_NAME:-ai-conformance-e2e-gcp}"

# Acquire GCP Project (Prow vs Local Dev Mode)
if [[ -n "${BOSKOS_HOST:-}" && "${RUN_IN_PROW:-false}" == "true" ]]; then
    echo "=== Acquiring GCP GPU Project from Boskos ==="
    export GOPATH="${HOME}/go"
    export PATH="${GOPATH}/bin:${PATH}"

    if ! command -v boskosctl &>/dev/null; then
        echo "Installing boskosctl..."
        go install sigs.k8s.io/boskos/cmd/boskosctl@latest
    fi

    GCP_PROJECT=$(boskosctl acquire --server "${BOSKOS_HOST}" --type "${BOSKOS_RESOURCE_TYPE}" --owner "${JOB_NAME}")
    export GCP_PROJECT
    echo "Acquired GCP Project: ${GCP_PROJECT}"

    # Start Boskos heartbeat in background
    (
        while true; do
            sleep 60
            boskosctl heartbeat --server "${BOSKOS_HOST}" --name "${GCP_PROJECT}" --owner "${JOB_NAME}" || break
        done
    ) &
    HEARTBEAT_PID=$!
else
    echo "=== Running in Local/Dev Mode ==="
    if [[ -z "${GCP_PROJECT:-}" ]]; then
        echo "Error: GCP_PROJECT environment variable must be set in local dev mode."
        exit 1
    fi
fi

# Cleanup handler on exit
cleanup() {
    local exit_code=$?
    echo "================================================================"
    echo "Cleaning up E2E test environment (Exit Code: ${exit_code})"
    echo "================================================================"

    if [[ -n "${HEARTBEAT_PID:-}" ]]; then
        kill "${HEARTBEAT_PID}" &>/dev/null || true
    fi

    if [[ -n "${GCP_PROJECT:-}" ]]; then
        echo "Deleting GCE VM ${VM_NAME} in project ${GCP_PROJECT}..."
        gcloud compute instances delete "${VM_NAME}" \
            --project="${GCP_PROJECT}" \
            --zone="${GCE_ZONE}" \
            --quiet || true

        if [[ -n "${BOSKOS_HOST:-}" && "${RUN_IN_PROW:-false}" == "true" ]]; then
            echo "Releasing Boskos GCP project ${GCP_PROJECT}..."
            boskosctl release --server "${BOSKOS_HOST}" --name "${GCP_PROJECT}" --owner "${JOB_NAME}" || true
        fi
    fi
}
trap cleanup EXIT

echo "================================================================"
echo "1. Creating GCE GPU Instance (${VM_NAME}) in ${GCE_ZONE}"
echo "================================================================"
gcloud compute instances create "${VM_NAME}" \
    --project="${GCP_PROJECT}" \
    --zone="${GCE_ZONE}" \
    --machine-type="${GCE_MACHINE_TYPE}" \
    --accelerator="${GCE_ACCELERATOR}" \
    --image-family="${GCE_IMAGE_FAMILY}" \
    --image-project="${GCE_IMAGE_PROJECT}" \
    --boot-disk-size=100GB \
    --maintenance-policy=TERMINATE

echo "Waiting for SSH to be available on ${VM_NAME}..."
for i in {1..30}; do
    if gcloud compute ssh "${VM_NAME}" --project="${GCP_PROJECT}" --zone="${GCE_ZONE}" --command="echo ready" &>/dev/null; then
        echo "SSH is ready!"
        break
    fi
    sleep 5
done

echo "================================================================"
echo "2. Setting up GPU-enabled Kubernetes (nvkind) on GCE VM"
echo "================================================================"
# Transfer codebase to VM
gcloud compute scp --project="${GCP_PROJECT}" --zone="${GCE_ZONE}" --recurse "${REPO_ROOT}" "${VM_NAME}:~/ai-conformance"

# Run setup & nvkind cluster creation on VM
gcloud compute ssh "${VM_NAME}" --project="${GCP_PROJECT}" --zone="${GCE_ZONE}" --command="bash -s" <<'REMOTE_SCRIPT'
set -euo pipefail

echo "Installing Docker & NVIDIA Container Toolkit..."
sudo apt-get update && sudo apt-get install -y docker.io helm
sudo usermod -aG docker $USER

echo "Installing Go & nvkind..."
wget -q https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
export PATH="/usr/local/go/bin:${PATH}"

go install github.com/NVIDIA/nvkind/cmd/nvkind@latest
go install sigs.k8s.io/kind@latest
export PATH="${HOME}/go/bin:${PATH}"

echo "Creating nvkind cluster..."
nvkind cluster create --name ai-conformance-cluster --k8s-version v1.35.0

echo "Cluster nodes & GPUs:"
kubectl get nodes -o wide
REMOTE_SCRIPT

echo "================================================================"
echo "3. Deploying Cluster Prerequisites (NVIDIA GPU Operator / DRA Driver)"
echo "================================================================"
gcloud compute ssh "${VM_NAME}" --project="${GCP_PROJECT}" --zone="${GCE_ZONE}" --command="bash -s" <<REMOTE_STACK
set -euo pipefail
export PATH="/usr/local/go/bin:\${HOME}/go/bin:\${PATH}"

echo "Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.2/cert-manager.yaml
kubectl rollout status deployment -n cert-manager cert-manager --timeout=5m

echo "Installing NVIDIA GPU Operator / DRA driver..."
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update
helm upgrade -i gpu-operator nvidia/gpu-operator \
    --namespace gpu-operator \
    --create-namespace \
    --set driver.enabled=false \
    --set toolkit.enabled=true \
    --wait --timeout 10m
REMOTE_STACK

echo "================================================================"
echo "4. Executing AI Conformance Test Suite (test/)"
echo "================================================================"
gcloud compute ssh "${VM_NAME}" --project="${GCP_PROJECT}" --zone="${GCE_ZONE}" --command="bash -s" <<'REMOTE_TEST'
set -euo pipefail
export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"

cd ~/ai-conformance
mkdir -p _artifacts

echo "Running go test ./test/..."
go test -v ./test/... \
    -accelerator-type=nvidia \
    -allocation-mode=auto \
    -json | tee _artifacts/results.json
REMOTE_TEST

# Copy test artifacts back to host
gcloud compute scp --project="${GCP_PROJECT}" --zone="${GCE_ZONE}" --recurse "${VM_NAME}:~/ai-conformance/_artifacts/*" "${ARTIFACTS}/"

echo "================================================================"
echo "AI Conformance E2E Test Run Completed Successfully!"
echo "================================================================"
