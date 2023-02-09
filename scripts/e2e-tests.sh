#!/bin/bash

set -e

# This scripts prepares a cluster where we install the kcrypt CRDs.
# This is where sealed volumes are created.

GINKGO_NODES="${GINKGO_NODES:-1}"
K3S_IMAGE="rancher/k3s:v1.26.1-k3s1"

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CLUSTER_NAME=$(echo $RANDOM | md5sum | head -c 10; echo;)
KUBECONFIG=$(mktemp)

# https://unix.stackexchange.com/a/423052
getFreePort() {
  echo $(comm -23 <(seq "30000" "30200" | sort) <(ss -Htan | awk '{print $4}' | cut -d':' -f2 | sort -u) | shuf | head -n "1")
}

cleanup() {
  echo "Cleaning up $CLUSTER_NAME"
  k3d cluster delete "$CLUSTER_NAME" || true
  rm -rf "$KUBECONFIG"
}
trap cleanup EXIT

# Create a cluster and bind ports 80 and 443 on the host
# This will allow us to access challenger server on 10.0.2.2 which is the IP
# on which qemu "sees" the host.
k3d cluster create "$CLUSTER_NAME" -p '80:80@server:0' -p '443:443@server:0' --image "$K3S_IMAGE"
k3d kubeconfig get "$CLUSTER_NAME" > "$KUBECONFIG"

# Build the docker image
IMG=controller:latest make docker-build

# Import the image to the cluster
k3d image import -c "$CLUSTER_NAME" controller:latest

# Install cert manager
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
kubectl wait --for=condition=Available deployment --timeout=2m -n cert-manager --all

# Replace the CLUSTER_IP in the kustomize resource
# Only needed for debugging so that we can access the server from the host
# (the 10.0.2.2 IP address is only useful from within qemu)
export CLUSTER_IP=$(docker inspect "k3d-${CLUSTER_NAME}-server-0"  | jq -r '.[0].NetworkSettings.Networks[].IPAddress')
envsubst \
    < "$SCRIPT_DIR/../tests/assets/challenger-server-ingress.template.yaml" \
    > "$SCRIPT_DIR/../tests/assets/challenger-server-ingress.yaml"

# Install the challenger server kustomization
kubectl apply -k "$SCRIPT_DIR/../tests/assets/"

# 10.0.2.2 is where the vm sees the host
# https://stackoverflow.com/a/6752280
export KMS_ADDRESS="10.0.2.2.challenger.sslip.io"

PATH=$PATH:$GOPATH/bin ginkgo --nodes $GINKGO_NODES --fail-fast -r ./tests/