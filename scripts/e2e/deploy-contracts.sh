#!/bin/bash

# This script deploys the required contracts to L1 and L2 chains
# It uses the endpoints configured by setup-devnet.sh

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Check dependencies
if ! command -v forge &> /dev/null; then
    echo "Error: forge is not installed. Please install Foundry from https://getfoundry.sh/"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed. Please install it using your package manager."
    exit 1
fi

# Check if endpoints file exists
if [ ! -f "devnet-endpoints.json" ]; then
    echo "Error: devnet-endpoints.json not found. Please run setup-devnet.sh first."
    exit 1
fi

# Load endpoints from JSON file
L1_RPC_URL=$(jq -r '.l1_rpc_url' devnet-endpoints.json)
L2_1_RPC_URL=$(jq -r '.l2_1_rpc_url' devnet-endpoints.json)
L2_2_RPC_URL=$(jq -r '.l2_2_rpc_url' devnet-endpoints.json)
L2_1_CHAIN_ID=$(jq -r '.l2_1_chain_id' devnet-endpoints.json)
L2_2_CHAIN_ID=$(jq -r '.l2_2_chain_id' devnet-endpoints.json)

# Use the first account from the default Kurtosis pre-funded accounts
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DEPLOYER_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

echo "Deploying contracts with account: $DEPLOYER_ADDRESS"

# Path to prover-contracts repo
PROVER_CONTRACTS_PATH="/Users/iannorden/go/src/github.com/polymerdao/prover-contracts"

# 1. Deploy Registry contract to L1
echo "Deploying Registry contract to L1..."
cd "$PROVER_CONTRACTS_PATH"
RESULT=$(forge script script/DeployRegistry.s.sol \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract Registry contract address
REGISTRY_ADDRESS=$(echo "$RESULT" | jq -r '.returns.registry.value')
echo "Registry deployed at: $REGISTRY_ADDRESS"

# 2. Deploy NativeProver contracts to L2-1
echo "Deploying NativeProver contracts to L2-1..."
RESULT=$(forge script script/DeployNativeProver.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract NativeProver contract address on L2-1
NATIVE_PROVER_L2_1_ADDRESS=$(echo "$RESULT" | jq -r '.returns.nativeProver.value')
echo "NativeProver on L2-1 deployed at: $NATIVE_PROVER_L2_1_ADDRESS"

# 3. Deploy NativeProver contracts to L2-2
echo "Deploying NativeProver contracts to L2-2..."
RESULT=$(forge script script/DeployNativeProver.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract NativeProver contract address on L2-2
NATIVE_PROVER_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.nativeProver.value')
echo "NativeProver on L2-2 deployed at: $NATIVE_PROVER_L2_2_ADDRESS"

# 4. Deploy test contract to L2-1 with a known storage value
echo "Deploying test contract to L2-1..."
RESULT=$(forge script script/DeployTestContract.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract test contract address
TEST_CONTRACT_ADDRESS=$(echo "$RESULT" | jq -r '.returns.testContract.value')
TEST_STORAGE_SLOT="0x0000000000000000000000000000000000000000000000000000000000000000"
echo "Test contract deployed on L2-1 at: $TEST_CONTRACT_ADDRESS"

# 5. Register L2 chains in the Registry on L1
echo "Registering L2-1 chain in Registry..."
L2_1_CONFIG_TYPE=1  # Assuming this is OPStackBedrock
L2_1_ADDRESSES="[$NATIVE_PROVER_L2_1_ADDRESS]"
L2_1_STORAGE_SLOTS="[0]"

cast send $REGISTRY_ADDRESS "registerL2Chain(uint256,uint8,address[],uint256[])" \
    $L2_1_CHAIN_ID $L2_1_CONFIG_TYPE "$L2_1_ADDRESSES" "$L2_1_STORAGE_SLOTS" \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY"

echo "Registering L2-2 chain in Registry..."
L2_2_CONFIG_TYPE=1  # Assuming this is OPStackBedrock
L2_2_ADDRESSES="[$NATIVE_PROVER_L2_2_ADDRESS]"
L2_2_STORAGE_SLOTS="[0]"

cast send $REGISTRY_ADDRESS "registerL2Chain(uint256,uint8,address[],uint256[])" \
    $L2_2_CHAIN_ID $L2_2_CONFIG_TYPE "$L2_2_ADDRESSES" "$L2_2_STORAGE_SLOTS" \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY"

# Save contract addresses for later use
cat > contract-addresses.json << EOF
{
  "registry": "$REGISTRY_ADDRESS",
  "native_prover_l2_1": "$NATIVE_PROVER_L2_1_ADDRESS",
  "native_prover_l2_2": "$NATIVE_PROVER_L2_2_ADDRESS",
  "test_contract": "$TEST_CONTRACT_ADDRESS",
  "test_storage_slot": "$TEST_STORAGE_SLOT"
}
EOF

echo "Contract deployment complete! Addresses saved to contract-addresses.json"