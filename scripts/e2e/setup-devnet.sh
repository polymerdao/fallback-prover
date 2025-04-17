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

echo "Creating configuration file for OP-Stack devnet..."

# Create Kurtosis config file
cat > op-stack-testnet.yaml << EOF
optimism_package:
  chains:
    # L1 Chain
    - participants: 
      - el_type: geth  # L1 execution client
      network_params:
        name: l1-chain
        network_id: 1234

    # First L2 chain
    - participants:
      - el_type: op-geth  # L2 sequencer
      network_params:
        name: l2-chain-1
        network_id: 12345
        
    # Second L2 chain
    - participants:
      - el_type: op-geth  # L2 sequencer
      network_params:
        name: l2-chain-2
        network_id: 12346
EOF

echo "Starting Kurtosis OP-Stack devnet..."
kurtosis run github.com/ethpandaops/optimism-package --args-file ./op-stack-testnet.yaml

# Get the service information
echo "Retrieving endpoint information from Kurtosis..."
SERVICES_INFO=$(kurtosis service ls --output json)

# Extract RPC URLs
L1_RPC_URL=$(echo "$SERVICES_INFO" | jq -r '.[] | select(.name == "l1-chain-executionclient") | .ports.rpc.url')
L2_1_RPC_URL=$(echo "$SERVICES_INFO" | jq -r '.[] | select(.name == "l2-chain-1-sequencer") | .ports.rpc.url')
L2_2_RPC_URL=$(echo "$SERVICES_INFO" | jq -r '.[] | select(.name == "l2-chain-2-sequencer") | .ports.rpc.url')

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