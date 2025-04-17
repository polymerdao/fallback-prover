#!/bin/bash

# This script generates and verifies a proof between two L2 chains
# It uses the fallback-prover tool to generate the proof and 
# cast to submit the proof to the NativeProver contract

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed. Please install it using your package manager."
    exit 1
fi

if ! command -v cast &> /dev/null; then
    echo "Error: cast is not installed. Please install Foundry from https://getfoundry.sh/"
    exit 1
fi

# Check if required files exist
if [ ! -f "devnet-endpoints.json" ]; then
    echo "Error: devnet-endpoints.json not found. Please run setup-devnet.sh first."
    exit 1
fi

if [ ! -f "contract-addresses.json" ]; then
    echo "Error: contract-addresses.json not found. Please run deploy-contracts.sh first."
    exit 1
fi

# Load endpoint information
L1_RPC_URL=$(jq -r '.l1_rpc_url' devnet-endpoints.json)
L2_1_RPC_URL=$(jq -r '.l2_1_rpc_url' devnet-endpoints.json)
L2_2_RPC_URL=$(jq -r '.l2_2_rpc_url' devnet-endpoints.json)
L2_1_CHAIN_ID=$(jq -r '.l2_1_chain_id' devnet-endpoints.json)
L2_2_CHAIN_ID=$(jq -r '.l2_2_chain_id' devnet-endpoints.json)

# Load contract addresses
REGISTRY_ADDRESS=$(jq -r '.registry' contract-addresses.json)
NATIVE_PROVER_L2_1_ADDRESS=$(jq -r '.native_prover_l2_1' contract-addresses.json)
NATIVE_PROVER_L2_2_ADDRESS=$(jq -r '.native_prover_l2_2' contract-addresses.json)
TEST_CONTRACT_ADDRESS=$(jq -r '.test_contract' contract-addresses.json)
TEST_STORAGE_SLOT=$(jq -r '.test_storage_slot' contract-addresses.json)

# Private key for testing (from Kurtosis default accounts)
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

# 1. Wait for finality
echo "Waiting for L1 blocks to be created and finalized..."
sleep 60

# 2. Get the current state value from the test contract
echo "Getting current storage value from test contract..."
STORAGE_VALUE=$(cast storage $TEST_CONTRACT_ADDRESS $TEST_STORAGE_SLOT --rpc-url $L2_1_RPC_URL)
echo "Current storage value: $STORAGE_VALUE"

# 3. Generate proof calldata using the fallback-prover tool
echo "Generating proof calldata..."
CALLDATA=$(go run "$ROOT_DIR/cmd/main.go" prove \
  --l1-http-path "$L1_RPC_URL" \
  --src-l2-rpc "$L2_1_RPC_URL" \
  --dst-l2-rpc "$L2_2_RPC_URL" \
  --src-l2-chain-id "$L2_1_CHAIN_ID" \
  --dst-l2-chain-id "$L2_2_CHAIN_ID" \
  --src-address "$TEST_CONTRACT_ADDRESS" \
  --src-storage-slot "$TEST_STORAGE_SLOT" \
  --registry-address "$REGISTRY_ADDRESS")

echo "Generated calldata: $CALLDATA"

# 4. Verify the proof by submitting it to the NativeProver on L2-2
echo "Submitting proof to NativeProver on L2-2..."
TX_HASH=$(cast send "$NATIVE_PROVER_L2_2_ADDRESS" "$CALLDATA" \
  --rpc-url "$L2_2_RPC_URL" \
  --private-key "$PRIVATE_KEY")

echo "Proof submitted in transaction: $TX_HASH"

# 5. Wait for transaction confirmation
echo "Waiting for transaction confirmation..."
cast receipt $TX_HASH --rpc-url $L2_2_RPC_URL

# 6. Verify that the proof was successful by checking for events
echo "Checking transaction status and events..."
EVENTS=$(cast receipt $TX_HASH --rpc-url $L2_2_RPC_URL)
SUCCESS=$(echo "$EVENTS" | grep "ProofVerified")

if [ -n "$SUCCESS" ]; then
  echo "✅ Proof verification successful!"
  echo "Storage value from L2-1 has been proven on L2-2"
else
  echo "❌ Proof verification may have failed. Check transaction logs for details."
  cast receipt $TX_HASH --rpc-url $L2_2_RPC_URL
fi