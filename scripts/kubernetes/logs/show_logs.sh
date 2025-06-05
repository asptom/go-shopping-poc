#!/bin/bash

# Replace <namespace> with your desired namespace
NAMESPACE="shopping"

echo ""
echo "-----------------------------------------------------------------------"
echo -n "Enter the namespace to get logs from (default: $NAMESPACE): "
read -r input_namespace
if [[ -n "$input_namespace" ]]; then
    NAMESPACE="$input_namespace"
fi
echo "Using namespace: $NAMESPACE"
echo "-----------------------------------------------------------------------"
echo ""

# Get the list of pod names in the specified namespace
PODS=$(kubectl get pods -n $NAMESPACE -o jsonpath='{.items[*].metadata.name}')

# Loop through each pod and get logs for all containers
for POD in $PODS; do
    echo "--- Logs for pod: $POD ---"
    kubectl logs $POD -n $NAMESPACE --all-containers
    echo "---------------------------"
    echo "" # Add a blank line for readability
done
