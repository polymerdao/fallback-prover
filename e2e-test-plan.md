# Fallback Prover E2E Testing Plan

This document outlines the approach for end-to-end testing of the Fallback Prover system using Kurtosis to set up a local OP-Stack testnet.

## Overview

We'll create a test environment with:
1. One L1 chain (Ethereum)
2. Two OP-Stack L2 chains that settle to the L1
3. Registry contract on L1
4. NativeProver contracts on both L2s
5. Test the generation and verification of proofs between L2s

## Automation Scripts

The entire process is automated using a set of shell scripts located in the `scripts/e2e` directory:

1. **e2e-test.sh**: Master script for running the entire E2E test flow
2. **setup-devnet.sh**: Configure and start Kurtosis environment
3. **deploy-contracts.sh**: Deploy all required contracts to L1 and L2s
4. **generate-and-verify-proof.sh**: Generate a proof and verify it

To run the E2E tests:

```bash
./scripts/e2e/e2e-test.sh
```

## Setup Steps

### 1. Kurtosis Configuration

Create a Kurtosis YAML configuration for a mini OP-Stack network:

```yaml
# op-stack-testnet.yaml
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
```

### 2. Deploy Kurtosis Network

```bash
kurtosis run github.com/ethpandaops/optimism-package --args-file ./op-stack-testnet.yaml
```

### 3. Contract Deployment

#### 3.1 Deploy Registry Contract to L1

1. Navigate to prover-contracts repository
2. Configure deployment to use the L1 RPC endpoint provided by Kurtosis
3. Deploy the Registry contract using Foundry:

```bash
cd /Users/iannorden/go/src/github.com/polymerdao/prover-contracts/
forge script script/DeployRegistry.s.sol --rpc-url <L1-RPC-URL> --private-key <PRIVATE_KEY> --broadcast
```

#### 3.2 Deploy NativeProver to L2s

1. Deploy NativeProver and associated contracts to both L2 chains:

```bash
cd /Users/iannorden/go/src/github.com/polymerdao/prover-contracts/
forge script script/DeployNativeProver.s.sol --rpc-url <L2-1-RPC-URL> --private-key <PRIVATE_KEY> --broadcast
forge script script/DeployNativeProver.s.sol --rpc-url <L2-2-RPC-URL> --private-key <PRIVATE_KEY> --broadcast
```

#### 3.3 Register L2 Chains in Registry

1. Register each L2 chain in the Registry contract with appropriate configuration:

```bash
cast send <REGISTRY_ADDRESS> "registerL2Chain(uint256,uint8,(address[],uint256[]))" <L2_CHAIN_ID> <CONFIG_TYPE> <ADDRESSES_AND_SLOTS> --rpc-url <L1-RPC-URL> --private-key <PRIVATE_KEY>
```

### 4. Test Data Setup

1. Deploy test contracts on L2-1 with known storage state
2. Setup test data that will be used for proof verification

### 5. Proof Generation

Generate proof using the fallback-prover tool:

```bash
cd /Users/iannorden/go/src/github.com/polymerdao/fallback-prover/
go run ./cmd/main.go prove \
  --l1-http-path <L1-RPC-URL> \
  --src-l2-rpc <L2-1-RPC-URL> \
  --dst-l2-rpc <L2-2-RPC-URL> \
  --src-l2-chain-id <L2_1_CHAIN_ID> \
  --dst-l2-chain-id <L2_2_CHAIN_ID> \
  --src-address <TEST_CONTRACT_ADDRESS> \
  --src-storage-slot <TEST_STORAGE_SLOT> \
  --registry-address <REGISTRY_ADDRESS>
```

### 6. Proof Verification

Verify the proof by sending the generated calldata to the NativeProver on L2-2:

```bash
cast send <NATIVE_PROVER_ADDRESS_ON_L2_2> "prove(...)" <CALLDATA> --rpc-url <L2-2-RPC-URL> --private-key <PRIVATE_KEY>
```

## Future Enhancements

1. Test various failure cases
2. Benchmark proof generation and verification times
3. Simulate network conditions and chain reorganizations
4. Test with different OP-Stack configurations