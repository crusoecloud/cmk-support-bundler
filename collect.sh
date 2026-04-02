#!/bin/bash
# CMK Support Bundler

set -eo pipefail

DEBUG_NAMESPACE="crusoe-debug"
MANIFEST_URL="https://raw.githubusercontent.com/crusoecloud/cmk-support-bundler/main/deploy/cmk-support-bundler.yaml"
OUTPUT="debug.zip"

echo "=== CMK Support Bundler ==="
echo ""

# Get the SlurmCluster namespace
SLURM_NAMESPACE="${1:-}"
if [ -z "$SLURM_NAMESPACE" ]; then
  read -p "Enter the namespace where your SlurmCluster is deployed: " SLURM_NAMESPACE
fi

if [ -z "$SLURM_NAMESPACE" ]; then
  echo "Error: namespace is required."
  exit 1
fi

echo ""
echo "This script will:"
echo "  1. Deploy a debug collection Job to the cluster"
echo "  2. Collect Slurm cluster diagnostics (logs, GPU, network, system info)"
echo "  3. Download the results as debug.zip"
echo "  4. Clean up all created resources"
echo ""
echo "Kubeconfig: ${KUBECONFIG:-$HOME/.kube/config}"
echo "Context:    $(kubectl config current-context)"
echo "Cluster:    $(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')"
echo ""
echo "Namespaces accessed (read-only):"
echo "  - $SLURM_NAMESPACE (SlurmCluster namespace — full diagnostics)"
echo "  - slinky (operator and topograph logs, helm releases)"
echo "  - cert-manager (pod listing, helm release)"
echo "  - nvidia-gpu-operator (pod listing)"
echo "  - nvidia-network-operator (pod listing)"
echo "  - crusoe-system (pod listing)"
echo "  - kube-system (pod listing)"
echo "  - crusoe-debug (created for the collection Job, cleaned up after)"
echo ""
echo "For details on what data is collected, see:"
echo "  https://github.com/crusoecloud/cmk-support-bundler#what-is-collected"
echo ""
echo "NOTE: No customer workloads, user data, or secrets are collected."
echo "      Only Slurm infrastructure diagnostics are gathered."
echo ""
read -p "Proceed? [y/N] " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 0
fi
echo ""

# Fetch manifest, replace namespace placeholder, and save locally
echo "Deploying debug job..."
RENDERED_MANIFEST=$(mktemp)
curl -sL "$MANIFEST_URL" | sed "s/__SLURM_NAMESPACE__/$SLURM_NAMESPACE/g" > "$RENDERED_MANIFEST"
kubectl apply -f "$RENDERED_MANIFEST"

# Wait for the pod to be running
echo "Waiting for collection pod to start..."
kubectl wait --for=condition=ready pod -l job-name=cmk-support-bundler -n "$DEBUG_NAMESPACE" --timeout=5m

# Wait for the output file to be written (poll every 10s)
echo "Waiting for collection to complete..."
POD=$(kubectl get pods -n "$DEBUG_NAMESPACE" -l job-name=cmk-support-bundler -o jsonpath='{.items[0].metadata.name}')
while ! kubectl exec -n "$DEBUG_NAMESPACE" "$POD" -- test -f /output/debug.zip 2>/dev/null; do
  sleep 10
done

# Download results
echo "Downloading results..."
kubectl cp "$DEBUG_NAMESPACE/$POD:/output/debug.zip" "./$OUTPUT"

# Cleanup
echo "Cleaning up..."
kubectl delete -f "$RENDERED_MANIFEST" --ignore-not-found
rm -f "$RENDERED_MANIFEST"

echo ""
echo "=== Done ==="
echo "Output: $OUTPUT"
echo ""
echo "View with:"
echo "  unzip $OUTPUT"
