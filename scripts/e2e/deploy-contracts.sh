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

# Verify that the endpoints are accessible
check_endpoint() {
  local url="$1"
  local name="$2"
  
  echo "Checking connectivity to $name at $url..."
  if ! curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' "$url" > /dev/null; then
    echo "Error: Cannot connect to $name at $url. Please check that the devnet is running properly."
    return 1
  fi
  echo "$name is accessible."
  return 0
}

if ! check_endpoint "$L1_RPC_URL" "L1 chain" || \
   ! check_endpoint "$L2_1_RPC_URL" "L2-1 chain" || \
   ! check_endpoint "$L2_2_RPC_URL" "L2-2 chain"; then
  echo "Error: Some endpoints are not accessible. Please restart the devnet."
  exit 1
fi

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

# 2. Deploy OPStackBedrockProver to L2-1
echo "Deploying OPStackBedrockProver to L2-1..."
RESULT=$(forge script script/DeployOPStackBedrockProver.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract OPStackBedrockProver contract address on L2-1
OP_STACK_BEDROCK_PROVER_L2_1_ADDRESS=$(echo "$RESULT" | jq -r '.returns.opStackBedrockProver.value')
echo "OPStackBedrockProver on L2-1 deployed at: $OP_STACK_BEDROCK_PROVER_L2_1_ADDRESS"

# 3. Deploy NativeProver contracts to L2-1
echo "Deploying NativeProver contracts to L2-1..."
RESULT=$(SETTLEMENT_REGISTRY=$REGISTRY_ADDRESS \
    L1_CHAIN_ID=1234 \
    BLOCK_HASH_ORACLE=$(cast call --rpc-url "$L2_1_RPC_URL" $(cast call --rpc-url "$L2_1_RPC_URL" 0x4200000000000000000000000000000000000015 "PORTAL()") "l1BlockHash()") \
    forge script script/DeployNativeProver.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract NativeProver contract address on L2-1
NATIVE_PROVER_L2_1_ADDRESS=$(echo "$RESULT" | jq -r '.returns.nativeProver.value')
echo "NativeProver on L2-1 deployed at: $NATIVE_PROVER_L2_1_ADDRESS"

# Deploy OPStackBedrockProver to L2-2
echo "Deploying OPStackBedrockProver to L2-2..."
RESULT=$(forge script script/DeployOPStackBedrockProver.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract OPStackBedrockProver contract address on L2-2
OP_STACK_BEDROCK_PROVER_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.opStackBedrockProver.value')
echo "OPStackBedrockProver on L2-2 deployed at: $OP_STACK_BEDROCK_PROVER_L2_2_ADDRESS"

# Deploy OPStackCannonProver to L2-2 (for demonstration)
echo "Deploying OPStackCannonProver to L2-2..."
RESULT=$(forge script script/DeployOPStackCannonProver.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract OPStackCannonProver contract address on L2-2
OP_STACK_CANNON_PROVER_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.opStackCannonProver.value')
echo "OPStackCannonProver on L2-2 deployed at: $OP_STACK_CANNON_PROVER_L2_2_ADDRESS"

# Deploy NativeProver contracts to L2-2
echo "Deploying NativeProver contracts to L2-2..."
RESULT=$(SETTLEMENT_REGISTRY=$REGISTRY_ADDRESS \
    L1_CHAIN_ID=1234 \
    BLOCK_HASH_ORACLE=$(cast call --rpc-url "$L2_2_RPC_URL" $(cast call --rpc-url "$L2_2_RPC_URL" 0x4200000000000000000000000000000000000015 "PORTAL()") "l1BlockHash()") \
    forge script script/DeployNativeProver.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract NativeProver contract address on L2-2
NATIVE_PROVER_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.nativeProver.value')
echo "NativeProver on L2-2 deployed at: $NATIVE_PROVER_L2_2_ADDRESS"

# 4. Deploy test contract to L2-1 with a known storage value
echo "Deploying test contract to L2-1..."
RESULT=$(TEST_CONTRACT_INITIAL_VALUE=123456789 \
    forge script script/DeployTestContract.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --json)

# Extract test contract address
TEST_CONTRACT_ADDRESS=$(echo "$RESULT" | jq -r '.returns.testContract.value')
TEST_STORAGE_SLOT="0x0000000000000000000000000000000000000000000000000000000000000000"
echo "Test contract deployed on L2-1 at: $TEST_CONTRACT_ADDRESS"

# 5. Update L2 chain configurations in the Registry on L1
echo "Configuring L2-1 chain in Registry..."
# First we need to grant permissions to the deployer for this chain ID
cast send $REGISTRY_ADDRESS "grantChainID(address,uint256)" \
    $DEPLOYER_ADDRESS $L2_1_CHAIN_ID \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY"

# Output Oracle address (Bedrock requires this to be the first address in the addresses array)
L2_OUTPUT_ORACLE_ADDRESS=$(cast call --rpc-url "$L2_1_RPC_URL" 0x4200000000000000000000000000000000000015 "L2_ORACLE()")
echo "L2-1 Output Oracle address: $L2_OUTPUT_ORACLE_ADDRESS"

# Create an ABI-encoded L2Configuration struct for L2-1
# Format: (address prover, address[] addresses, uint256[] storageSlots, uint256 versionNumber, uint256 finalityDelaySeconds, uint8 l2Type)
L2_1_CONFIG=$(cast abi-encode "Config(address,address[],uint256[],uint256,uint256,uint8)" \
    $OP_STACK_BEDROCK_PROVER_L2_1_ADDRESS \
    "[$L2_OUTPUT_ORACLE_ADDRESS]" \
    "[3]" \
    0 \
    0 \
    1)  # Type 1 = OPStackBedrock

cast send $REGISTRY_ADDRESS "updateL2ChainConfiguration(uint256,(address,address[],uint256[],uint256,uint256,uint8))" \
    $L2_1_CHAIN_ID \
    $L2_1_CONFIG \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY"

echo "Configuring L2-2 chain in Registry..."
# Grant permissions for this chain ID
cast send $REGISTRY_ADDRESS "grantChainID(address,uint256)" \
    $DEPLOYER_ADDRESS $L2_2_CHAIN_ID \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY"

# Output Oracle address for L2-2
L2_OUTPUT_ORACLE_ADDRESS_2=$(cast call --rpc-url "$L2_2_RPC_URL" 0x4200000000000000000000000000000000000015 "L2_ORACLE()")
echo "L2-2 Output Oracle address: $L2_OUTPUT_ORACLE_ADDRESS_2"

# Create an ABI-encoded L2Configuration struct for L2-2
L2_2_CONFIG=$(cast abi-encode "Config(address,address[],uint256[],uint256,uint256,uint8)" \
    $OP_STACK_BEDROCK_PROVER_L2_2_ADDRESS \
    "[$L2_OUTPUT_ORACLE_ADDRESS_2]" \
    "[3]" \
    0 \
    0 \
    1)  # Type 1 = OPStackBedrock

cast send $REGISTRY_ADDRESS "updateL2ChainConfiguration(uint256,(address,address[],uint256[],uint256,uint256,uint8))" \
    $L2_2_CHAIN_ID \
    $L2_2_CONFIG \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY"

# Save contract addresses for later use
cat > contract-addresses.json << EOF
{
  "registry": "$REGISTRY_ADDRESS",
  "native_prover_l2_1": "$NATIVE_PROVER_L2_1_ADDRESS",
  "native_prover_l2_2": "$NATIVE_PROVER_L2_2_ADDRESS",
  "op_stack_bedrock_prover_l2_1": "$OP_STACK_BEDROCK_PROVER_L2_1_ADDRESS",
  "op_stack_bedrock_prover_l2_2": "$OP_STACK_BEDROCK_PROVER_L2_2_ADDRESS",
  "op_stack_cannon_prover_l2_2": "$OP_STACK_CANNON_PROVER_L2_2_ADDRESS",
  "test_contract": "$TEST_CONTRACT_ADDRESS",
  "test_storage_slot": "$TEST_STORAGE_SLOT",
  "l2_1_chain_id": $L2_1_CHAIN_ID,
  "l2_2_chain_id": $L2_2_CHAIN_ID
}
EOF

echo "Contract deployment complete! Addresses saved to contract-addresses.json"