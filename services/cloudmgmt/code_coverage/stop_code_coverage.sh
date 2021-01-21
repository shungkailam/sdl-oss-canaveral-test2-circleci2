#!/usr/bin/env sh

# Helper script to stop code coverage gracefully and generate a report

set -o errexit
set -o pipefail
set -o nounset
# set -o xtrace

if [ $# -ne 2 ]; then
    echo "Please specify namespace and component ONLY.\nSample Usage: ./stop_code_coverage.sh test cloudmgmt"
fi

NAMESPACE=$1
COMPONENT=$2

# Get pod name
POD="$(kubectl get pods -n $NAMESPACE | grep -i $COMPONENT | cut -d' ' -f1)"
echo "Pod Name: $POD"

# Get PID of running code coverage binary
PID="$(kubectl exec -it $POD -- ps aux | sed 1d | grep -i $COMPONENT | awk '{print $1}')"
echo "$COMPONENT PID: $PID"

# Send SIGUSR1 to code coverage binary
echo "Sending SIGUSR1 to PID: $PID"
kubectl exec -it $POD -- kill -SIGUSR1 $PID

echo "Stopped code coverage binary." 
