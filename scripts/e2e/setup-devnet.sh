#!/bin/bash

# This script sets up a local OP-Stack devnet using Kurtosis
# It creates one L1 and two L2 chains for e2e testing

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Check dependencies
if ! command -v kurtosis &> /dev/null; then
    echo "Error: kurtosis is not installed. Please install it from https://docs.kurtosis.com/install/"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed. Please install it using your package manager."
    exit 1
fi

echo "Using existing OP-Stack devnet configuration..."

# Config file is already created and versioned in the repository
# Modify it if needed at op-stack-testnet.yaml

echo "Starting Kurtosis OP-Stack devnet..."
kurtosis run github.com/ethpandaops/optimism-package --args-file ./op-stack-testnet.yaml

# Get the service information
echo "Retrieving endpoint information from Kurtosis..."
# List running enclaves and find the one with our services
echo "Searching for the correct Kurtosis enclave..."
RUNNING_ENCLAVES=$(kurtosis enclave ls | grep RUNNING | awk '{print $2}')

# Initialize variables
ENCLAVE_NAME=""
ENCLAVE_INFO=""

# Loop through each enclave to find the one with our services
for enclave in $RUNNING_ENCLAVES; do
  echo "Checking enclave: $enclave"
  # Get the enclave info
  CURRENT_INFO=$(kurtosis enclave inspect "$enclave")

  # Check if this enclave has our OP-Stack services
  if echo "$CURRENT_INFO" | grep -q "op-el-12345\|op-el-12346"; then
    echo "Found OP-Stack services in enclave: $enclave"
    ENCLAVE_NAME="$enclave"
    ENCLAVE_INFO="$CURRENT_INFO"
    break
  fi
done

if [ -z "$ENCLAVE_NAME" ]; then
  echo "Error: Could not find any Kurtosis enclave with OP-Stack services. Make sure the devnet is running."
  exit 1
fi
echo "Using Kurtosis enclave: $ENCLAVE_NAME"

# Store the raw output for parsing
SERVICE_OUTPUT="$ENCLAVE_INFO"

# Print all available services for debugging
echo "Available Kurtosis services:"
echo "$SERVICE_OUTPUT"

# Extract RPC endpoints directly using awk patterns
echo "Extracting RPC endpoints from service output..."

# Direct extraction for L1 using awk
L1_PORT=$(echo "$SERVICE_OUTPUT" | awk '/el-1-geth-teku/{flag=1;next}/^[a-zA-Z0-9]/{if(flag==1)flag=0}/rpc:/{if(flag==1){if(match($0,/127\.0\.0\.1:[0-9]+/)){print substr($0,RSTART+10,RLENGTH-10);exit}}}')
if [ -n "$L1_PORT" ]; then
  L1_RPC_URL="http://localhost:$L1_PORT"
  echo "Found L1 RPC at port $L1_PORT"
else
  echo "Could not find L1 RPC port"
  L1_RPC_URL=""
fi

# Direct extraction for L2-1 using awk
L2_1_PORT=$(echo "$SERVICE_OUTPUT" | awk '/op-el-12345/{flag=1;next}/^[a-zA-Z0-9]/{if(flag==1)flag=0}/rpc:/{if(flag==1){if(match($0,/127\.0\.0\.1:[0-9]+/)){print substr($0,RSTART+10,RLENGTH-10);exit}}}')
if [ -n "$L2_1_PORT" ]; then
  L2_1_RPC_URL="http://localhost:$L2_1_PORT"
  echo "Found L2-1 RPC at port $L2_1_PORT"
else
  echo "Could not find L2-1 RPC port"
  L2_1_RPC_URL=""
fi

# Direct extraction for L2-2
L2_2_PORT=$(echo "$SERVICE_OUTPUT" | awk '/op-el-12346/{flag=1;next}/^[a-zA-Z0-9]/{if(flag==1)flag=0}/rpc:/{if(flag==1){if(match($0,/127\.0\.0\.1:[0-9]+/)){print substr($0,RSTART+10,RLENGTH-10);exit}}}')
if [ -n "$L2_2_PORT" ]; then
  L2_2_RPC_URL="http://localhost:$L2_2_PORT"
  echo "Found L2-2 RPC at port $L2_2_PORT"
else
  echo "Could not find L2-2 RPC port"
  L2_2_RPC_URL=""
fi

echo "Endpoints found:"
echo "L1 RPC URL: $L1_RPC_URL"
echo "L2-1 RPC URL: $L2_1_RPC_URL"
echo "L2-2 RPC URL: $L2_2_RPC_URL"

# Verify that we have valid URLs
validate_url() {
  local url="$1"
  local name="$2"

  if [ -z "$url" ]; then
    echo "Error: Could not extract $name URL from Kurtosis services" >&2
    echo "Available services output:" >&2
    echo "$SERVICE_OUTPUT" | head -30 >&2
    return 1
  fi

  # Basic URL validation
  if ! [[ "$url" =~ ^https?:// ]]; then
    echo "Warning: $name URL ($url) does not start with http:// or https://" >&2
  fi

  return 0
}

# Function to test RPC endpoint connectivity
test_rpc_connection() {
  local url="$1"
  local name="$2"

  echo "Testing connectivity to $name at $url..."

  # Try to get chain ID
  local result=$(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    "$url")

  if [ $? -ne 0 ] || [ -z "$result" ]; then
    echo "ERROR: Could not connect to $name at $url" >&2
    return 1
  fi

  local chain_id=$(echo "$result" | jq -r '.result')
  if [ -z "$chain_id" ] || [ "$chain_id" = "null" ]; then
    echo "WARNING: Could not get chain ID from $name at $url" >&2
    echo "Response: $result" >&2
    return 1
  fi

  echo "$name is accessible. Chain ID: $chain_id"
  return 0
}

# We need ALL endpoints for the script to succeed
if validate_url "$L1_RPC_URL" "L1 RPC" &&
   validate_url "$L2_1_RPC_URL" "L2-1 RPC" &&
   validate_url "$L2_2_RPC_URL" "L2-2 RPC"; then
  echo "Devnet setup complete!"
  echo "L1 RPC URL: $L1_RPC_URL"
  echo "L2-1 RPC URL: $L2_1_RPC_URL"
  echo "L2-2 RPC URL: $L2_2_RPC_URL"

  # Save the endpoints to a file for later use
  cat > devnet-endpoints.json << EOF
{
  "l1_rpc_url": "$L1_RPC_URL",
  "l2_1_rpc_url": "$L2_1_RPC_URL",
  "l2_2_rpc_url": "$L2_2_RPC_URL",
  "l2_1_chain_id": 12345,
  "l2_2_chain_id": 12346
}
EOF

  echo "Endpoint information saved to devnet-endpoints.json"

  # If TEST_CONNECTIONS is set, test connectivity to all endpoints
  if [ "${TEST_CONNECTIONS:-}" = "1" ]; then
    echo "Testing connectivity to all endpoints..."
    test_rpc_connection "$L1_RPC_URL" "L1 chain"
    test_rpc_connection "$L2_1_RPC_URL" "L2-1 chain"
    test_rpc_connection "$L2_2_RPC_URL" "L2-2 chain"
  fi
else
  echo "Failed to extract all required endpoints from Kurtosis." >&2
  echo "Please check the Kurtosis logs and make sure the services are running properly." >&2
  echo "Available services:" >&2
  echo "$SERVICE_OUTPUT" >&2

  # Show what we did find for debugging
  echo "Partial results:"
  echo "L1 RPC URL: $L1_RPC_URL"
  echo "L2-1 RPC URL: $L2_1_RPC_URL"
  echo "L2-2 RPC URL: $L2_2_RPC_URL"

  exit 1
fi
