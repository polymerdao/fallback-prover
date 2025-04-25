#!/bin/bash

# This script deploys the required contracts to L1 and L2 chains
# It uses the endpoints configured by setup-devnet.sh
# All deployments use CREATE2 for deterministic addresses across chains

# Exit immediately if a command fails and print commands as they're executed
set -e

# Disable Etherscan access entirely
export FOUNDRY_OFFLINE=true

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Optional: Run check-create2-addresses.sh to compute and display expected addresses
if [ "${CHECK_CREATE2_ADDRESSES:-}" = "1" ]; then
    echo "Computing expected CREATE2 addresses before deployment..."
    "$SCRIPT_DIR/check-create2-addresses.sh"
    echo "Press Enter to continue with deployment, or Ctrl+C to abort."
    read -r
fi

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
export PRIVATE_KEY
DEPLOYER_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

echo "Deploying contracts with account: $DEPLOYER_ADDRESS"

# Path to prover-contracts repo
PROVER_CONTRACTS_PATH="/Users/iannorden/go/src/github.com/polymerdao/prover-contracts"

# 1. Deploy Registry contract to L1 using CREATE2
echo "Deploying Registry contract to L1 using CREATE2..."
cd "$PROVER_CONTRACTS_PATH"
echo "Working directory: $(pwd)"
echo "Checking for dependencies..."
ls -la lib

echo "Running forge script to deploy Registry with CREATE2..."
# Use only one forge script command with higher gas price and priority fee
set -x
RESULT=$(forge script script/DeployRegistryCreate2.s.sol \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)
SCRIPT_EXIT_CODE=$?
set +x

# If forge script fails, show detailed error and exit
if [ $SCRIPT_EXIT_CODE -ne 0 ]; then
    echo "ERROR: forge script failed with exit code $SCRIPT_EXIT_CODE"
    exit 1
fi

echo "Script exit code: $SCRIPT_EXIT_CODE"

# Extract Registry contract address
REGISTRY_ADDRESS=$(echo "$RESULT" | jq -r '.returns.registry.value // empty')
if [ -z "$REGISTRY_ADDRESS" ]; then
    echo "ERROR: Failed to extract Registry address from output"
    echo "Raw output: $RESULT"
    exit 1
fi

echo "Registry deployed at: $REGISTRY_ADDRESS"

# Add a delay between transactions to ensure nonce increments properly
echo "Waiting for transaction confirmation..."
sleep 5

# 2. Deploy OPStackCannonProver to L2-1 using CREATE2 for deterministic address (since we're using Cannon, not Bedrock)
echo "Deploying OPStackCannonProver to L2-1 using CREATE2..."
RESULT=$(forge script script/DeployOPStackCannonProverCreate2.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)

# Extract OPStackCannonProver contract address on L2-1
OP_STACK_CANNON_PROVER_L2_1_ADDRESS=$(echo "$RESULT" | jq -r '.returns.opStackCannonProver.value // empty')
if [ -z "$OP_STACK_CANNON_PROVER_L2_1_ADDRESS" ]; then
    echo "ERROR: Failed to extract OPStackCannonProver L2-1 address from output"
    echo "Raw output: $RESULT"
    exit 1
fi
echo "OPStackCannonProver on L2-1 deployed at: $OP_STACK_CANNON_PROVER_L2_1_ADDRESS"

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

# 3. Deploy NativeProver contracts to L2-1 using CREATE2
echo "Deploying NativeProver contracts to L2-1 using CREATE2..."
RESULT=$(SETTLEMENT_REGISTRY=$REGISTRY_ADDRESS \
    L1_CHAIN_ID=1234 \
    BLOCK_HASH_ORACLE=0x4200000000000000000000000000000000000015 \
    L2_CONFIG_MAPPING_SLOT=1 \
    L1_CONFIG_MAPPING_SLOT=2 \
    SETTLEMENT_BLOCKS_DELAY=10 \
    forge script script/DeployNativeProverCreate2.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)

# Extract NativeProver contract address on L2-1
NATIVE_PROVER_L2_1_ADDRESS=$(echo "$RESULT" | jq -r '.returns.nativeProver.value // empty')
if [ -z "$NATIVE_PROVER_L2_1_ADDRESS" ]; then
    echo "ERROR: Failed to extract NativeProver L2-1 address from output"
    echo "Raw output: $RESULT"
    exit 1
fi
echo "NativeProver on L2-1 deployed at: $NATIVE_PROVER_L2_1_ADDRESS"

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

# Skip deploying OPStackBedrockProver to L2-2 since we're using Cannon

# Deploy OPStackCannonProver to L2-2 using CREATE2 for deterministic address
echo "Deploying OPStackCannonProver to L2-2 using CREATE2..."
RESULT=$(forge script script/DeployOPStackCannonProverCreate2.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)

# Extract OPStackCannonProver contract address on L2-2
OP_STACK_CANNON_PROVER_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.opStackCannonProver.value // empty')
if [ -z "$OP_STACK_CANNON_PROVER_L2_2_ADDRESS" ]; then
    echo "ERROR: Failed to extract OPStackCannonProver L2-2 address from output"
    echo "Raw output: $RESULT"
    exit 1
fi
echo "OPStackCannonProver on L2-2 deployed at: $OP_STACK_CANNON_PROVER_L2_2_ADDRESS"

# Verify that both addresses match (they should, due to CREATE2)
if [ "$OP_STACK_CANNON_PROVER_L2_1_ADDRESS" == "$OP_STACK_CANNON_PROVER_L2_2_ADDRESS" ]; then
    echo "✅ SUCCESS: OPStackCannonProver addresses match across both L2 chains!"
    echo "Deterministic address (CREATE2): $OP_STACK_CANNON_PROVER_L2_1_ADDRESS"
else
    echo "❌ ERROR: OPStackCannonProver addresses do not match across L2 chains!"
    echo "L2-1: $OP_STACK_CANNON_PROVER_L2_1_ADDRESS"
    echo "L2-2: $OP_STACK_CANNON_PROVER_L2_2_ADDRESS"
    echo "This indicates an issue with the CREATE2 deployment"
fi

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

# Deploy NativeProver contracts to L2-2 using CREATE2
echo "Deploying NativeProver contracts to L2-2 using CREATE2..."
RESULT=$(SETTLEMENT_REGISTRY=$REGISTRY_ADDRESS \
    L1_CHAIN_ID=1234 \
    BLOCK_HASH_ORACLE=0x4200000000000000000000000000000000000015 \
    L2_CONFIG_MAPPING_SLOT=1 \
    L1_CONFIG_MAPPING_SLOT=2 \
    SETTLEMENT_BLOCKS_DELAY=10 \
    forge script script/DeployNativeProverCreate2.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)

# Extract NativeProver contract address on L2-2
NATIVE_PROVER_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.nativeProver.value // empty')
if [ -z "$NATIVE_PROVER_L2_2_ADDRESS" ]; then
    echo "ERROR: Failed to extract NativeProver L2-2 address from output"
    echo "Raw output: $RESULT"
    exit 1
fi
echo "NativeProver on L2-2 deployed at: $NATIVE_PROVER_L2_2_ADDRESS"

# Verify that both addresses match (they should, due to CREATE2)
if [ "$NATIVE_PROVER_L2_1_ADDRESS" == "$NATIVE_PROVER_L2_2_ADDRESS" ]; then
    echo "✅ SUCCESS: NativeProver addresses match across both L2 chains!"
    echo "Deterministic address (CREATE2): $NATIVE_PROVER_L2_1_ADDRESS"
else
    echo "❌ ERROR: NativeProver addresses do not match across L2 chains!"
    echo "L2-1: $NATIVE_PROVER_L2_1_ADDRESS"
    echo "L2-2: $NATIVE_PROVER_L2_2_ADDRESS"
    echo "This indicates an issue with the CREATE2 deployment parameters"
fi

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

# 4. Deploy test contract to L2-1 with a known storage value using CREATE2
echo "Deploying test contract to L2-1 using CREATE2..."
RESULT=$(TEST_CONTRACT_INITIAL_VALUE=123456789 \
    forge script script/DeployTestContractCreate2.s.sol \
    --rpc-url "$L2_1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)

# Extract test contract address
TEST_CONTRACT_ADDRESS=$(echo "$RESULT" | jq -r '.returns.testContract.value // empty')
if [ -z "$TEST_CONTRACT_ADDRESS" ]; then
    echo "ERROR: Failed to extract test contract address from output"
    echo "Raw output: $RESULT"
    exit 1
fi
TEST_STORAGE_SLOT="0x0000000000000000000000000000000000000000000000000000000000000000"
echo "Test contract deployed on L2-1 at: $TEST_CONTRACT_ADDRESS"

# Deploy the same test contract to L2-2 to verify CREATE2 works
echo "Deploying test contract to L2-2 using CREATE2 (should have same address)..."
RESULT=$(TEST_CONTRACT_INITIAL_VALUE=123456789 \
    forge script script/DeployTestContractCreate2.s.sol \
    --rpc-url "$L2_2_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --broadcast \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy \
    --json)

# Extract test contract address on L2-2
TEST_CONTRACT_L2_2_ADDRESS=$(echo "$RESULT" | jq -r '.returns.testContract.value // empty')
if [ -z "$TEST_CONTRACT_L2_2_ADDRESS" ]; then
    echo "ERROR: Failed to extract test contract L2-2 address from output"
    echo "Raw output: $RESULT"
    exit 1
fi

# Verify that test contract addresses match
if [ "$TEST_CONTRACT_ADDRESS" == "$TEST_CONTRACT_L2_2_ADDRESS" ]; then
    echo "✅ SUCCESS: TestContract addresses match across both L2 chains!"
    echo "Deterministic address (CREATE2): $TEST_CONTRACT_ADDRESS"
else
    echo "❌ ERROR: TestContract addresses do not match across L2 chains!"
    echo "L2-1: $TEST_CONTRACT_ADDRESS"
    echo "L2-2: $TEST_CONTRACT_L2_2_ADDRESS"
    echo "This indicates an issue with the CREATE2 deployment"
fi

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

# 5. Update L2 chain configurations in the Registry on L1
echo "Configuring L2-1 chain in Registry..."
# First we need to grant permissions to the deployer for this chain ID
# Temporarily disable Etherscan API warning for this command
echo "Granting permissions to the deployer for chain ID $L2_1_CHAIN_ID..."

echo "DEBUG: Registry address: $REGISTRY_ADDRESS"
echo "DEBUG: Deployer address: $DEPLOYER_ADDRESS"
echo "DEBUG: L2_1_CHAIN_ID: $L2_1_CHAIN_ID"
echo "DEBUG: L1_RPC_URL: $L1_RPC_URL"

set -x
CAST_OUTPUT=$(FOUNDRY_VERBOSE=true cast send $REGISTRY_ADDRESS "grantChainID(address,uint256)" \
    $DEPLOYER_ADDRESS $L2_1_CHAIN_ID \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy 2>&1)
CAST_EXIT_CODE=$?
set +x

echo "DEBUG: CAST_OUTPUT: $CAST_OUTPUT"
echo "DEBUG: CAST_EXIT_CODE: $CAST_EXIT_CODE"

if [ $CAST_EXIT_CODE -ne 0 ]; then
    echo "ERROR: Failed to grant chain ID permissions to the deployer. Exit code: $CAST_EXIT_CODE"
    echo "Output: $CAST_OUTPUT"
    exit 1
fi
echo "Successfully granted permissions for chain ID $L2_1_CHAIN_ID"

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

# Since we're using Cannon, we need the DisputeGameFactory contract addresses from L1
# Extract these addresses programmatically from the Kurtosis deployment
echo "Extracting DisputeGameFactory addresses from Kurtosis deployment..."

# Get the name of the current active enclave
ENCLAVE_NAME=$(kurtosis enclave ls | grep RUNNING | awk '{print $2}' | head -1)
if [ -z "$ENCLAVE_NAME" ]; then
    echo "ERROR: No running Kurtosis enclaves found. Make sure the devnet is running."
    exit 1
fi
echo "Using Kurtosis enclave: $ENCLAVE_NAME"

# Extract the DisputeGameFactory address for L2-1 (chain ID 12345) using proper regex to get the Ethereum address
DISPUTE_GAME_FACTORY_ADDRESS_L2_1=$(kurtosis service inspect "$ENCLAVE_NAME" op-challenger-12345 | grep -o "game-factory-address=0x[a-fA-F0-9]\{40\}" | cut -d= -f2)
if [ -z "$DISPUTE_GAME_FACTORY_ADDRESS_L2_1" ]; then
    echo "ERROR: Failed to extract DisputeGameFactory address for L2-1"
    echo "Using fallback address 0x3096f75001e2d9132efedcaf50b54c4f1fcd152c"
    DISPUTE_GAME_FACTORY_ADDRESS_L2_1="0x3096f75001e2d9132efedcaf50b54c4f1fcd152c"
fi

# Extract the DisputeGameFactory address for L2-2 (chain ID 12346) using proper regex to get the Ethereum address
DISPUTE_GAME_FACTORY_ADDRESS_L2_2=$(kurtosis service inspect "$ENCLAVE_NAME" op-challenger-12346 | grep -o "game-factory-address=0x[a-fA-F0-9]\{40\}" | cut -d= -f2)
if [ -z "$DISPUTE_GAME_FACTORY_ADDRESS_L2_2" ]; then
    echo "ERROR: Failed to extract DisputeGameFactory address for L2-2"
    echo "Using fallback address 0xe0efde4efa0a8a249749bddb05e2f2c89b79515c"
    DISPUTE_GAME_FACTORY_ADDRESS_L2_2="0xe0efde4efa0a8a249749bddb05e2f2c89b79515c"
fi

echo "Using DisputeGameFactory address for L2-1: $DISPUTE_GAME_FACTORY_ADDRESS_L2_1"
echo "Using DisputeGameFactory address for L2-2: $DISPUTE_GAME_FACTORY_ADDRESS_L2_2"

# Define the required storage slots for Cannon prover
DISPUTE_GAME_FACTORY_LIST_SLOT=104
FAULT_DISPUTE_GAME_ROOT_CLAIM_SLOT="0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ad1"
L2_FAULT_DISPUTE_GAME_STATUS_SLOT=0

echo "Updating L2-1 chain configuration in Registry..."

# First, generate the calldata for the transaction
echo "Generating calldata for updateL2ChainConfiguration..."
STRUCT_ARG="($OP_STACK_CANNON_PROVER_L2_1_ADDRESS,[$DISPUTE_GAME_FACTORY_ADDRESS_L2_1],[$DISPUTE_GAME_FACTORY_LIST_SLOT,$FAULT_DISPUTE_GAME_ROOT_CLAIM_SLOT,$L2_FAULT_DISPUTE_GAME_STATUS_SLOT],0,0,2)"
L2_CONFIG_CALLDATA=$(cast calldata "updateL2ChainConfiguration(uint256,(address,address[],uint256[],uint256,uint256,uint8))" $L2_1_CHAIN_ID "$STRUCT_ARG")
echo "Generated calldata: $L2_CONFIG_CALLDATA"

# Now send the transaction with the pre-encoded calldata
set -x
CAST_OUTPUT=$(cast send $REGISTRY_ADDRESS $L2_CONFIG_CALLDATA \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy 2>&1)
CAST_EXIT_CODE=$?
set +x

echo "DEBUG: CAST_OUTPUT: $CAST_OUTPUT"
echo "DEBUG: CAST_EXIT_CODE: $CAST_EXIT_CODE"

if [ $CAST_EXIT_CODE -ne 0 ]; then
    echo "ERROR: Failed to update L2-1 chain configuration. Exit code: $CAST_EXIT_CODE"
    echo "Output: $CAST_OUTPUT"
    exit 1
fi
echo "Successfully updated L2-1 chain configuration"

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

echo "Configuring L2-2 chain in Registry..."
# Grant permissions for this chain ID
echo "Granting permissions to the deployer for chain ID $L2_2_CHAIN_ID..."
set -x
CAST_OUTPUT=$(FOUNDRY_VERBOSE=true cast send $REGISTRY_ADDRESS "grantChainID(address,uint256)" \
    $DEPLOYER_ADDRESS $L2_2_CHAIN_ID \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy 2>&1)
CAST_EXIT_CODE=$?
set +x

echo "DEBUG: CAST_OUTPUT: $CAST_OUTPUT"
echo "DEBUG: CAST_EXIT_CODE: $CAST_EXIT_CODE"

if [ $CAST_EXIT_CODE -ne 0 ]; then
    echo "ERROR: Failed to grant chain ID permissions to the deployer. Exit code: $CAST_EXIT_CODE"
    echo "Output: $CAST_OUTPUT"
    exit 1
fi
echo "Successfully granted permissions for chain ID $L2_2_CHAIN_ID"

# Add delay between transactions
echo "Waiting for transaction confirmation..."
sleep 5

echo "Updating L2-2 chain configuration in Registry..."

# First, generate the calldata for the transaction
echo "Generating calldata for L2-2 updateL2ChainConfiguration..."
STRUCT_ARG="($OP_STACK_CANNON_PROVER_L2_2_ADDRESS,[$DISPUTE_GAME_FACTORY_ADDRESS_L2_2],[$DISPUTE_GAME_FACTORY_LIST_SLOT,$FAULT_DISPUTE_GAME_ROOT_CLAIM_SLOT,$L2_FAULT_DISPUTE_GAME_STATUS_SLOT],0,0,2)"
L2_CONFIG_CALLDATA=$(cast calldata "updateL2ChainConfiguration(uint256,(address,address[],uint256[],uint256,uint256,uint8))" $L2_2_CHAIN_ID "$STRUCT_ARG")
echo "Generated calldata: $L2_CONFIG_CALLDATA"

# Now send the transaction with the pre-encoded calldata
set -x
CAST_OUTPUT=$(cast send $REGISTRY_ADDRESS $L2_CONFIG_CALLDATA \
    --rpc-url "$L1_RPC_URL" \
    --private-key "$PRIVATE_KEY" \
    --gas-price 50000000000 \
    --priority-gas-price 3000000000 \
    --legacy 2>&1)
CAST_EXIT_CODE=$?
set +x

echo "DEBUG: CAST_OUTPUT: $CAST_OUTPUT"
echo "DEBUG: CAST_EXIT_CODE: $CAST_EXIT_CODE"

if [ $CAST_EXIT_CODE -ne 0 ]; then
    echo "ERROR: Failed to update L2-2 chain configuration. Exit code: $CAST_EXIT_CODE"
    echo "Output: $CAST_OUTPUT"
    exit 1
fi
echo "Successfully updated L2-2 chain configuration"

# Add final delay to ensure all transactions are processed
echo "Waiting for final transaction confirmation..."
sleep 10

# Verify all variables are set and not null before saving
if [ -z "$REGISTRY_ADDRESS" ] || [ -z "$NATIVE_PROVER_L2_1_ADDRESS" ] || [ -z "$NATIVE_PROVER_L2_2_ADDRESS" ] || \
   [ -z "$OP_STACK_CANNON_PROVER_L2_1_ADDRESS" ] || [ -z "$OP_STACK_CANNON_PROVER_L2_2_ADDRESS" ] || \
   [ -z "$TEST_CONTRACT_ADDRESS" ] || [ -z "$L2_1_CHAIN_ID" ] || [ -z "$L2_2_CHAIN_ID" ]; then
  echo "ERROR: One or more contract addresses are not set. Cannot create contract-addresses.json"
  exit 1
fi

# Save contract addresses for later use
cat > contract-addresses.json << EOF
{
  "registry": "$REGISTRY_ADDRESS",
  "native_prover_l2_1": "$NATIVE_PROVER_L2_1_ADDRESS",
  "native_prover_l2_2": "$NATIVE_PROVER_L2_2_ADDRESS",
  "op_stack_cannon_prover_l2_1": "$OP_STACK_CANNON_PROVER_L2_1_ADDRESS",
  "op_stack_cannon_prover_l2_2": "$OP_STACK_CANNON_PROVER_L2_2_ADDRESS",
  "dispute_game_factory_l2_1": "$DISPUTE_GAME_FACTORY_ADDRESS_L2_1",
  "dispute_game_factory_l2_2": "$DISPUTE_GAME_FACTORY_ADDRESS_L2_2",
  "test_contract": "$TEST_CONTRACT_ADDRESS",
  "test_storage_slot": "$TEST_STORAGE_SLOT",
  "l2_1_chain_id": $L2_1_CHAIN_ID,
  "l2_2_chain_id": $L2_2_CHAIN_ID,
  "dispute_game_factory_list_slot": $DISPUTE_GAME_FACTORY_LIST_SLOT,
  "fault_dispute_game_root_claim_slot": "$FAULT_DISPUTE_GAME_ROOT_CLAIM_SLOT",
  "l2_fault_dispute_game_status_slot": $L2_FAULT_DISPUTE_GAME_STATUS_SLOT
}
EOF

echo "Contract deployment complete! Addresses saved to contract-addresses.json"
