package provers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/polymerdao/fallback_prover/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

var _ ISettledStateProver = &OPStackBedrockProver{}

// OPStackBedrockProver handles proof generation for OP Stack Bedrock chains
type OPStackBedrockProver struct {
	l1Client          IEthClient
	l1RPC             IRPCClient
	l2Client          IEthClient
	l2RPC             IRPCClient
	abi               abi.ABI
	l2OutputOracleABI abi.ABI
}

// NewOPStackBedrockProver creates a new prover instance for OP Stack Bedrock
func NewOPStackBedrockProver(l1Client IEthClient, l1RPC IRPCClient, l2Client IEthClient, l2RPC IRPCClient) (*OPStackBedrockProver, error) {
	abiObj, err := getOPStackBedrockProverABI()
	if err != nil {
		return nil, err
	}

	l2OutputAbiObj, err := getL2OutputOracleABI()
	if err != nil {
		return nil, err
	}

	return &OPStackBedrockProver{
		l1Client:          l1Client,
		l1RPC:             l1RPC,
		l2Client:          l2Client,
		l2RPC:             l2RPC,
		abi:               abiObj,
		l2OutputOracleABI: l2OutputAbiObj,
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

var (
	L2MessagePasserAddress = common.HexToAddress("0x4200000000000000000000000000000000000016") // Standard address on OP Stack
)

func (p *OPStackBedrockProver) FindLatestResolved(ctx context.Context, config *types.L2ConfigInfo) (*big.Int, common.Address, error) {
	if len(config.Addresses) == 0 || len(config.StorageSlots) == 0 {
		return nil, common.Address{}, fmt.Errorf("invalid config: addresses or slots are empty")
	}

	latestOutputIndexData, err := p.l2OutputOracleABI.Pack("latestOutputIndex")
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to pack getL2OutputIndexAfter: %w", err)
	}

	l2OutputOracleAddr := config.Addresses[0]

	latestOutputIndexResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &l2OutputOracleAddr,
		Data: latestOutputIndexData,
	}, nil)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to call getL2OutputIndexAfter: %w", err)
	}

	latestOutputIndex := new(big.Int).SetBytes(latestOutputIndexResult)
	if latestOutputIndex.Cmp(big.NewInt(0)) < 0 {
		return nil, common.Address{}, fmt.Errorf("invalid latestOutputIndex: %s", latestOutputIndex.String())
	}

	return latestOutputIndex, l2OutputOracleAddr, nil
}

// GenerateSettledStateProof creates a proof for an OPStack Bedrock L2 against L1
func (p *OPStackBedrockProver) GenerateSettledStateProof(
	ctx context.Context,
	l1BlockNumber *big.Int, outputIndex *big.Int,
	l2OutputOracleAddr common.Address,
	config *types.L2ConfigInfo) ([]byte, *types2.Header, error) {
	if len(config.Addresses) == 0 || len(config.StorageSlots) == 0 {
		return nil, nil, fmt.Errorf("invalid config: addresses or slots are empty")
	}

	l2OutputData, err := p.l2OutputOracleABI.Pack("getL2Output", outputIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack getL2Output: %w", err)
	}

	l2OutputResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &l2OutputOracleAddr,
		Data: l2OutputData,
	}, l1BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call getL2Output: %w", err)
	}

	// OutputProposal struct has 3 fields: outputRoot, timestamp, l2BlockNumber
	type OutputProposal struct {
		OutputRoot    common.Hash
		Timestamp     *big.Int
		L2BlockNumber *big.Int
	}
	// First try to unpack directly into a struct
	var outputProposal OutputProposal

	// If the direct unpack fails, try the byte-by-byte approach
	if len(l2OutputResult) >= 96 { // 32 bytes for outputRoot, 32 bytes for timestamp, 32 bytes for l2BlockNumber
		copy(outputProposal.OutputRoot[:], l2OutputResult[:32])
		outputProposal.Timestamp = new(big.Int).SetBytes(l2OutputResult[32:64])
		outputProposal.L2BlockNumber = new(big.Int).SetBytes(l2OutputResult[64:96])
	} else {
		// Only try the ABI unpacking as a fallback
		if err := p.l2OutputOracleABI.UnpackIntoInterface(&outputProposal, "getL2Output", l2OutputResult); err != nil {
			return nil, nil, fmt.Errorf("failed to unpack output proposal: %w", err)
		}
	}

	l2Block, err := p.l2Client.BlockByNumber(ctx, outputProposal.L2BlockNumber) // nil means latest block
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest L2 block: %w", err)
	}

	// Use eth_getProof to get L2toL1MessagePasser info at the L2 output height
	var messagePasserProof types.StorageProofResult
	// Empty array for storage keys - we only need the account proof with storageHash
	err = p.l2RPC.CallContext(
		ctx,
		&messagePasserProof,
		"eth_getProof",
		L2MessagePasserAddress.Hex(),
		[]string{}, // No specific storage keys needed, we just want the storageHash
		toBlockNumArg(outputProposal.L2BlockNumber),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get message passer proof: %w", err)
	}

	// The storageHash from the proof is the L2ToL1MessagePasser root we need
	messagePasserRoot := messagePasserProof.StorageHash

	// Get the storage proof for the output proposal
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
		return nil, nil, fmt.Errorf("no storage slots provided in config")
	}

	// Get the storage proof from the L1 node
	var proof types.StorageProofResult
	err = p.l1RPC.CallContext(
		ctx,
		&proof,
		"eth_getProof",
		l2OutputOracleAddr.Hex(),
		[]string{storageSlot.Hex()},
		toBlockNumArg(l1BlockNumber))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get output oracle proof: %w", err)
	}

	l1StorageProof := make([][]byte, len(proof.StorageProof[0].Proof))
	for i, p := range proof.StorageProof[0].Proof {
		l1StorageProof[i] = common.FromHex(p)
	}

	account := Account{
		Nonce:    uint64(*proof.Nonce),
		Balance:  proof.Balance.ToInt(),
		Root:     proof.StorageHash,
		CodeHash: proof.CodeHash.Bytes(),
	}

	rlpEncodedOutputOracleData, err := rlp.EncodeToBytes(account)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to RLP encode account: %w", err)
	}

	l1AccountProof := make([][]byte, len(proof.AccountProof))
	for i, p := range proof.AccountProof {
		l1AccountProof[i] = common.FromHex(p)
	}

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
		return nil, nil, fmt.Errorf("failed to RLP encode settled state proof: %w", err)
	}

	return settledStateProof, l2Block.Header(), nil
}
