#!/bin/bash

# This script runs the ComputeCreate2Addresses script to get expected addresses
# This is a utility script that can be run before deployment to verify addresses

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../.." &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Check dependencies
if ! command -v forge &> /dev/null; then
    echo "Error: forge is not installed. Please install Foundry from https://getfoundry.sh/"
    exit 1
fi

# Path to prover-contracts repo
PROVER_CONTRACTS_PATH="/Users/iannorden/go/src/github.com/polymerdao/prover-contracts"
if [ ! -d "$PROVER_CONTRACTS_PATH" ]; then
    echo "Error: prover-contracts directory not found at $PROVER_CONTRACTS_PATH"
    exit 1
fi

echo "Computing expected CREATE2 addresses..."
cd "$PROVER_CONTRACTS_PATH"

# Use the default private key for address computation
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
DEPLOYER_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

echo "Using deployer address: $DEPLOYER_ADDRESS"
export PRIVATE_KEY

# Run the script to compute addresses
forge script script/ComputeCreate2Addresses.s.sol