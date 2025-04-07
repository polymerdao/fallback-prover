# Native Proof CLI Tool

This is a Golang CLI tool that generates calldata for transactions calling the NativeProver.prove() function, which is defined in the [prover-contracts](https://github.com/polymerdao/prover-contracts/contracts/core/native_fallback/L2/NativeProver.sol) repository.

## Overview

The native-proof tool allows you to generate proof calldata that can be used to verify the state of a contract on one L2 chain from another L2 chain. This is useful for cross-L2 communication and verification.

The tool supports OP Stack chains (both Bedrock and Cannon) and handles all the necessary RPC calls to build a valid proof structure.

## Building

To build the binary:

```bash
make build
```

This will create the binary at `./bin/native-proof`.

## Usage

The tool offers a `prove` command with the following required arguments:

```bash
./bin/native-proof prove \
  --src-l2-chain-id <source-l2-chain-id> \
  --dst-l2-chain-id <destination-l2-chain-id> \
  --src-l2-rpc <source-l2-rpc-url> \
  --dst-l2-rpc <destination-l2-rpc-url> \
  --src-address <source-contract-address> \
  --src-storage-slot <source-storage-slot> \
  --l1-rpc <l1-rpc-url>
```

### Parameters

- `src-l2-chain-id`: Chain ID of the source L2 chain where the contract is deployed
- `dst-l2-chain-id`: Chain ID of the destination L2 chain that will verify the proof
- `src-l2-rpc`: RPC URL for the source L2 chain
- `dst-l2-rpc`: RPC URL for the destination L2 chain
- `src-address`: Address of the contract on the source L2 chain
- `src-storage-slot`: Storage slot to prove in the contract
- `l1-rpc`: RPC URL for the L1 chain (Ethereum)
- `registry-address`: (Optional) Address of the Registry contract on L1

### Example

```bash
./bin/native-proof prove \
  --src-l2-chain-id 10 \
  --dst-l2-chain-id 42161 \
  --src-l2-rpc https://mainnet.optimism.io \
  --dst-l2-rpc https://arb1.arbitrum.io/rpc \
  --src-address 0x1234567890abcdef1234567890abcdef12345678 \
  --src-storage-slot 0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890 \
  --l1-rpc https://ethereum.publicnode.com
```

## How It Works

The tool performs the following steps:

1. Queries the Registry contract on L1 to get the configuration for both source and destination L2 chains
2. Gets the L1 block hash oracle address for the destination L2 chain
3. Retrieves the current L1 header hash from the destination L2 chain
4. Gets the L1 block corresponding to that hash
5. Generates a settled state proof based on the source L2 chain type (OPStackBedrock or OPStackCannon)
6. Creates a storage proof for the source contract address and storage slot
7. Packages everything into the calldata format expected by the NativeProver.prove() function

## License

[License terms]