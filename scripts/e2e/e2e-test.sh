#!/bin/bash

# Master script for running the entire E2E test flow

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

echo "========== Fallback Prover E2E Test =========="
echo "This script will:"
echo "1. Set up a local OP-Stack devnet using Kurtosis"
echo "2. Deploy Registry and NativeProver contracts (all using CREATE2 for deterministic addresses)"
echo "3. Generate and verify a proof between L2 chains"
echo "4. Clean up the environment"
echo "=============================================="
echo "Environment options:"
echo "- To check CREATE2 addresses before deployment: CHECK_CREATE2_ADDRESSES=1 ./e2e-test.sh"
echo "=============================================="

# Make all scripts executable
chmod +x "$SCRIPT_DIR/setup-devnet.sh"
chmod +x "$SCRIPT_DIR/deploy-contracts.sh"
chmod +x "$SCRIPT_DIR/generate-and-verify-proof.sh"
chmod +x "$SCRIPT_DIR/check-create2-addresses.sh"

# Set environment options
export CHECK_CREATE2_ADDRESSES="${CHECK_CREATE2_ADDRESSES:-0}"

# Step 1: Set up the devnet
echo -e "\n[Step 1] Setting up OP-Stack devnet using Kurtosis..."
"$SCRIPT_DIR/setup-devnet.sh"

# Step 2: Deploy contracts
echo -e "\n[Step 2] Deploying contracts to L1 and L2 chains..."
"$SCRIPT_DIR/deploy-contracts.sh"

# Step 3: Generate and verify proof
echo -e "\n[Step 3] Generating and verifying proof between L2 chains..."
"$SCRIPT_DIR/generate-and-verify-proof.sh"

# Step 4: Clean up (optional - uncomment to enable)
# echo -e "\n[Step 4] Cleaning up resources..."
# kurtosis enclave clean optimism_package

echo -e "\nE2E test completed successfully!"
echo "Devnet is still running. To clean up, run: kurtosis enclave clean optimism_package"