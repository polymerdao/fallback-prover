package provers

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/polymerdao/fallback_prover/types"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// OPStackBedrockProver handles proof generation for OP Stack Bedrock chains
type OPStackBedrockProver struct {
	l1Client IEthClient
	l1RPC    IRPCClient
	l2Client IEthClient
	l2RPC    IRPCClient
	abi      abi.ABI
}

// NewOPStackBedrockProver creates a new prover instance for OP Stack Bedrock
func NewOPStackBedrockProver(l1Client IEthClient, l1RPC IRPCClient, l2Client IEthClient, l2RPC IRPCClient) (*OPStackBedrockProver, error) {
	abiObj, err := getOPStackBedrockProverABI()
	if err != nil {
		return nil, err
	}

	return &OPStackBedrockProver{
		l1Client: l1Client,
		l1RPC:    l1RPC,
		l2Client: l2Client,
		l2RPC:    l2RPC,
		abi:      abiObj,
	}, nil
}

// getOPStackBedrockProverABI loads and parses the OPStackBedrockProver ABI from file
func getOPStackBedrockProverABI() (abi.ABI, error) {
	// Get the absolute path of the current file
	_, thisFile, _, _ := runtime.Caller(0)
	// Construct the path to the ABI file
	abiPath := filepath.Join(filepath.Dir(thisFile), "abis", "OPStackBedrockProver.abi.json")

	// Read the ABI file
	abiFile, err := os.Open(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to open OPStackBedrockProver ABI file: %w", err)
	}
	defer abiFile.Close()

	abiBytes, err := io.ReadAll(abiFile)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read OPStackBedrockProver ABI file: %w", err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse OPStackBedrockProver ABI: %w", err)
	}

	return parsedABI, nil
}

// getL2OutputOracleABI loads and returns the ABI for the L2OutputOracle contract
func getL2OutputOracleABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(`[
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "_l2BlockNumber",
					"type": "uint256"
				}
			],
			"name": "getL2OutputIndexAfter",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "_l2OutputIndex",
					"type": "uint256"
				}
			],
			"name": "getL2Output",
			"outputs": [
				{
					"components": [
						{
							"internalType": "bytes32",
							"name": "outputRoot",
							"type": "bytes32"
						},
						{
							"internalType": "uint128",
							"name": "timestamp",
							"type": "uint128"
						},
						{
							"internalType": "uint128",
							"name": "l2BlockNumber",
							"type": "uint128"
						}
					],
					"internalType": "struct Types.OutputProposal",
					"name": "",
					"type": "tuple"
				}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`))
}

// Constants for L2ToL1MessagePasser contract in OP Stack
const (
	L2MessagePasserAddress = "0x4200000000000000000000000000000000000016" // Standard address on OP Stack
)

// GenerateSettledStateProof creates a proof for an OPStack Bedrock L2 against L1
func (p *OPStackBedrockProver) GenerateSettledStateProof(
	ctx context.Context,
	config *types.L2ConfigInfo,
	l1BlockHash common.Hash,
) ([]byte, common.Hash, []byte, error) {
	if len(config.Addresses) == 0 || len(config.StorageSlots) == 0 {
		return nil, common.Hash{}, nil, fmt.Errorf("invalid config: addresses or slots are empty")
	}

	// Get the L2OutputOracle address from the config
	l2OutputOracleAddr := config.Addresses[0]

	// Step 1: Get the latest L2 block
	l2Block, err := p.l2Client.BlockByNumber(ctx, nil) // nil means latest block
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to get latest L2 block: %w", err)
	}

	l2BlockNumber := l2Block.Number()
	l2Header := l2Block.Header()
	l2StateRoot := l2Header.Root

	// Get the RLP encoded L2 header
	rlpEncodedL2Header, err := rlp.EncodeToBytes(l2Header)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode L2 header: %w", err)
	}

	// Step 2: Get the L2 message passer root using eth_getProof
	// Get the storage root (storageHash) of the L2ToL1MessagePasser contract
	messagePasserAddr := common.HexToAddress(L2MessagePasserAddress)

	// Use eth_getProof to get the account state proof including the storage root
	var messagePasserProof types.StorageProofResult
	// Empty array for storage keys - we only need the account proof with storageHash
	err = p.l2RPC.CallContext(
		ctx,
		&messagePasserProof,
		"eth_getProof",
		messagePasserAddr.Hex(),
		[]string{}, // No specific storage keys needed, we just want the storageHash
		toBlockNumArg(l2BlockNumber),
	)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to get message passer proof: %w", err)
	}

	// The storageHash from the proof is the L2ToL1MessagePasser root we need
	messagePasserRoot := messagePasserProof.StorageHash

	// Step 3: Get the L2 output index for the L2 block using L2OutputOracle on L1
	outputOracleABI, err := getL2OutputOracleABI()
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to parse L2OutputOracle ABI: %w", err)
	}

	outputIndexData, err := outputOracleABI.Pack("getL2OutputIndexAfter", l2BlockNumber)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to pack getL2OutputIndexAfter: %w", err)
	}

	outputIndexResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &l2OutputOracleAddr,
		Data: outputIndexData,
	}, nil)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to call getL2OutputIndexAfter: %w", err)
	}

	// Properly handle the unpacking - the value is a uint256 directly in the result bytes
	outputIndex := new(big.Int).SetBytes(outputIndexResult)
	// Check for errors or zero index
	if outputIndex.Cmp(big.NewInt(0)) < 0 {
		return nil, common.Hash{}, nil, fmt.Errorf("invalid output index: %s", outputIndex.String())
	}

	// Step 4: Get the output proposal for the L2 output index
	outputData, err := outputOracleABI.Pack("getL2Output", outputIndex)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to pack getL2Output: %w", err)
	}

	outputResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &l2OutputOracleAddr,
		Data: outputData,
	}, nil)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to call getL2Output: %w", err)
	}

	// OutputProposal struct has 3 fields: outputRoot, timestamp, l2BlockNumber
	var outputProposal struct {
		OutputRoot    common.Hash
		Timestamp     *big.Int
		L2BlockNumber *big.Int
	}
	if err := outputOracleABI.UnpackIntoInterface(&outputProposal, "getL2Output", outputResult); err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to unpack output proposal: %w", err)
	}

	// Step 5: Get the storage proof for the output proposal
	// Calculate the storage slot for the output index
	var storageSlot common.Hash
	if len(config.StorageSlots) > 0 {
		// Use the storage slot from the config
		baseSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[0])))
		storageSlot = crypto.Keccak256Hash(
			common.LeftPadBytes(outputIndex.Bytes(), 32),
			common.LeftPadBytes(baseSlot.Bytes(), 32),
		)
	} else {
		return nil, common.Hash{}, nil, fmt.Errorf("no storage slots provided in config")
	}

	// Get the storage proof from the L1 node
	var proof types.StorageProofResult
	err = p.l1RPC.CallContext(ctx, &proof, "eth_getProof", l2OutputOracleAddr.Hex(), []string{storageSlot.Hex()}, "latest")
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to get output oracle proof: %w", err)
	}

	// Convert storage proof to bytes
	l1StorageProof := make([][]byte, len(proof.StorageProof[0].Proof))
	for i, p := range proof.StorageProof[0].Proof {
		l1StorageProof[i] = common.FromHex(p)
	}

	// Create RLP encoded account data
	account := Account{
		Nonce:    uint64(*proof.Nonce),
		Balance:  proof.Balance.ToInt(),
		Root:     proof.StorageHash,
		CodeHash: proof.CodeHash.Bytes(),
	}

	rlpEncodedOutputOracleData, err := rlp.EncodeToBytes(account)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode account: %w", err)
	}

	// Convert account proof to bytes
	l1AccountProof := make([][]byte, len(proof.AccountProof))
	for i, p := range proof.AccountProof {
		l1AccountProof[i] = common.FromHex(p)
	}

	// Step 6: Package everything together in the format expected by the prover contract
	outputIndexBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(outputIndexBytes[24:32], outputIndex.Uint64())

	// Format: [l2MessagePasserStateRoot, outputIndex, l1StorageProof, rlpEncodedOutputOracleData, l1AccountProof]
	settledStateProofData := []interface{}{
		messagePasserRoot,
		outputIndexBytes,
		l1StorageProof,
		rlpEncodedOutputOracleData,
		l1AccountProof,
	}

	// RLP encode the final proof
	settledStateProof, err := rlp.EncodeToBytes(settledStateProofData)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode settled state proof: %w", err)
	}

	return settledStateProof, l2StateRoot, rlpEncodedL2Header, nil
}
