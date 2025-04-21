package provers

import (
	"context"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// OPStackCannonProver handles proof generation for OP Stack Cannon chains
type OPStackCannonProver struct {
	l1Client IEthClient
	l1RPC    IRPCClient
	l2Client IEthClient
	l2RPC    IRPCClient
	abi      abi.ABI
}

// NewOPStackCannonProver creates a new prover instance for OP Stack Cannon
func NewOPStackCannonProver(l1Client IEthClient, l1RPC IRPCClient, l2Client IEthClient, l2RPC IRPCClient) (*OPStackCannonProver, error) {
	abiObj, err := getOPStackCannonProverABI()
	if err != nil {
		return nil, err
	}

	return &OPStackCannonProver{
		l1Client: l1Client,
		l1RPC:    l1RPC,
		l2Client: l2Client,
		l2RPC:    l2RPC,
		abi:      abiObj,
	}, nil
}

// getOPStackCannonProverABI loads and parses the OPStackCannonProver ABI from file
func getOPStackCannonProverABI() (abi.ABI, error) {
	// Get the absolute path of the current file
	_, thisFile, _, _ := runtime.Caller(0)
	// Construct the path to the ABI file
	abiPath := filepath.Join(filepath.Dir(thisFile), "abis", "OPStackCannonProver.abi.json")

	// Read the ABI file
	abiFile, err := os.Open(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to open OPStackCannonProver ABI file: %w", err)
	}
	defer abiFile.Close()

	abiBytes, err := io.ReadAll(abiFile)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read OPStackCannonProver ABI file: %w", err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse OPStackCannonProver ABI: %w", err)
	}

	return parsedABI, nil
}

// getDisputeGameFactoryABI returns the ABI for the DisputeGameFactory contract
func getDisputeGameFactoryABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(`[
		{
			"inputs": [
				{
					"internalType": "uint32",
					"name": "_gameType",
					"type": "uint32"
				},
				{
					"internalType": "bytes32",
					"name": "_rootClaim",
					"type": "bytes32"
				},
				{
					"internalType": "uint256",
					"name": "_expectedBlockNumber",
					"type": "uint256"
				}
			],
			"name": "create",
			"outputs": [
				{
					"internalType": "contract IDisputeGame",
					"name": "",
					"type": "address"
				}
			],
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "gameCount",
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
					"name": "_index",
					"type": "uint256"
				}
			],
			"name": "gameAtIndex",
			"outputs": [
				{
					"internalType": "contract IDisputeGame",
					"name": "",
					"type": "address"
				}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`))
}

// getFaultDisputeGameABI returns the ABI for the FaultDisputeGame contract
func getFaultDisputeGameABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(`[
		{
			"inputs": [],
			"name": "rootClaim",
			"outputs": [
				{
					"internalType": "bytes32",
					"name": "",
					"type": "bytes32"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "status",
			"outputs": [
				{
					"internalType": "enum GameStatus",
					"name": "",
					"type": "uint8"
				}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`))
}

// Constants for OP Stack Cannon
const (
	// Standard address for L2ToL1MessagePasser in OP Stack
	CannonL2MessagePasserAddress = "0x4200000000000000000000000000000000000016"
)

// GenerateSettledStateProof creates a proof for an OPStack Cannon L2 against L1
func (p *OPStackCannonProver) GenerateSettledStateProof(
	ctx context.Context,
	config *types.L2ConfigInfo) ([]byte, common.Hash, []byte, error) {
	if len(config.Addresses) < 1 || len(config.StorageSlots) < 3 {
		return nil, common.Hash{}, nil, fmt.Errorf("invalid config: addresses or slots are insufficient")
	}

	// Get addresses and slots from the config
	disputeGameFactoryAddr := config.Addresses[0]
	disputeGameFactoryListSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[0])))
	faultDisputeGameRootClaimSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[1])))
	faultDisputeGameStatusSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[2])))

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

	// Step: 2 Get the message passer state root using eth_getProof
	messagePasserAddr := common.HexToAddress(CannonL2MessagePasserAddress)

	// Get the storage root (storageHash) of the L2ToL1MessagePasser contract
	var messagePasserProof types.StorageProofResult
	err = p.l2RPC.CallContext(
		ctx,
		&messagePasserProof,
		"eth_getProof",
		messagePasserAddr.Hex(),
		[]string{}, // No specific storage keys needed
		toBlockNumArg(l2BlockNumber),
	)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to get message passer proof: %w", err)
	}

	messagePasserRoot := messagePasserProof.StorageHash

	// Step 3: Get the latest block hash from L2
	latestBlockHash := l2Block.Hash()

	// Step 4: Interact with the DisputeGameFactory to get information about dispute games
	disputeGameFactoryABI, err := getDisputeGameFactoryABI()
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to parse DisputeGameFactory ABI: %w", err)
	}

	// Get the total number of games
	gameCountData, err := disputeGameFactoryABI.Pack("gameCount")
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to pack gameCount call: %w", err)
	}
	gameCountResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &disputeGameFactoryAddr,
		Data: gameCountData,
	}, nil)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to call gameCount: %w", err)
	}

	// Handle the case where we receive raw bytes or properly ABI-encoded data
	var gameCount *big.Int

	// First, try to parse it as raw bytes
	if len(gameCountResult) == 32 {
		gameCount = new(big.Int).SetBytes(gameCountResult)
		fmt.Printf("Parsed gameCount from bytes: %v (len: %d, bytes: %x)\n", gameCount, len(gameCountResult), gameCountResult)
	} else if len(gameCountResult) > 0 {
		// If we have non-empty data but not 32 bytes, try ABI unpacking
		if err := disputeGameFactoryABI.UnpackIntoInterface(&gameCount, "gameCount", gameCountResult); err != nil {
			fmt.Printf("Failed to unpack gameCount, got error: %v (result len: %d, data: %x)\n", err, len(gameCountResult), gameCountResult)
			return nil, common.Hash{}, nil, fmt.Errorf("failed to unpack game count: %w", err)
		}
	} else {
		fmt.Printf("Received empty gameCountResult (len: %d)\n", len(gameCountResult))
		return nil, common.Hash{}, nil, fmt.Errorf("empty game count result from contract")
	}

	// For simplicity, we'll use the latest game (in a real implementation, we'd need to find the specific game for our block)
	gameIndex := new(big.Int).Sub(gameCount, big.NewInt(1))
	if gameIndex.Sign() < 0 {
		return nil, common.Hash{}, nil, fmt.Errorf("no dispute games found")
	}

	// Get the game address
	gameAtIndexData, err := disputeGameFactoryABI.Pack("gameAtIndex", gameIndex)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to pack gameAtIndex call: %w", err)
	}

	gameAtIndexResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &disputeGameFactoryAddr,
		Data: gameAtIndexData,
	}, nil)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to call gameAtIndex: %w", err)
	}

	var gameAddress common.Address

	// First check if we got a valid result
	if len(gameAtIndexResult) == 32 {
		// The address is in the last 20 bytes of a 32-byte value
		copy(gameAddress[:], gameAtIndexResult[12:]) // Take last 20 bytes
		fmt.Printf("Parsed gameAddress from bytes: %s (len: %d, bytes: %x)\n", gameAddress.Hex(), len(gameAtIndexResult), gameAtIndexResult)
	} else if len(gameAtIndexResult) > 0 {
		// Try to unpack via ABI
		if err := disputeGameFactoryABI.UnpackIntoInterface(&gameAddress, "gameAtIndex", gameAtIndexResult); err != nil {
			fmt.Printf("Failed to unpack gameAddress, got error: %v (result len: %d, data: %x)\n", err, len(gameAtIndexResult), gameAtIndexResult)
			return nil, common.Hash{}, nil, fmt.Errorf("failed to unpack game address: %w", err)
		}
	} else {
		fmt.Printf("Received empty gameAtIndexResult (len: %d)\n", len(gameAtIndexResult))
		return nil, common.Hash{}, nil, fmt.Errorf("empty game address result from contract")
	}

	// Step 5: Get storage proof for the dispute game factory
	// Calculate the storage slot for the game index
	gameIndexSlot := crypto.Keccak256Hash(
		common.LeftPadBytes(gameIndex.Bytes(), 32),
		common.LeftPadBytes(disputeGameFactoryListSlot.Bytes(), 32),
	)

	// Get the storage proof from L1 for the dispute game factory
	var disputeGameFactoryProof types.StorageProofResult
	err = p.l1RPC.CallContext(
		ctx,
		&disputeGameFactoryProof,
		"eth_getProof",
		disputeGameFactoryAddr.Hex(),
		[]string{gameIndexSlot.Hex()},
		"latest",
	)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to get dispute game factory proof: %w", err)
	}

	// Convert storage proof to bytes
	disputeFaultGameStorageProof := make([][]byte, len(disputeGameFactoryProof.StorageProof[0].Proof))
	for i, p := range disputeGameFactoryProof.StorageProof[0].Proof {
		disputeFaultGameStorageProof[i] = common.FromHex(p)
	}

	// Create RLP encoded dispute game factory account
	disputeGameFactoryAccount := Account{
		Nonce:    uint64(*disputeGameFactoryProof.Nonce),
		Balance:  disputeGameFactoryProof.Balance.ToInt(),
		Root:     disputeGameFactoryProof.StorageHash,
		CodeHash: disputeGameFactoryProof.CodeHash.Bytes(),
	}

	rlpEncodedDisputeGameFactoryData, err := rlp.EncodeToBytes(disputeGameFactoryAccount)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode dispute game factory account: %w", err)
	}

	// Convert account proof to bytes
	disputeGameFactoryAccountProof := make([][]byte, len(disputeGameFactoryProof.AccountProof))
	for i, p := range disputeGameFactoryProof.AccountProof {
		disputeGameFactoryAccountProof[i] = common.FromHex(p)
	}

	// Step 6: Get information from the FaultDisputeGame contract
	faultDisputeGameABI, err := getFaultDisputeGameABI()
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to parse FaultDisputeGame ABI: %w", err)
	}

	// Get the root claim
	rootClaimData, err := faultDisputeGameABI.Pack("rootClaim")
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to pack rootClaim call: %w", err)
	}

	rootClaimResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &gameAddress,
		Data: rootClaimData,
	}, nil)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to call rootClaim: %w", err)
	}

	var rootClaim common.Hash

	// First check if we got a valid result
	if len(rootClaimResult) == 32 {
		copy(rootClaim[:], rootClaimResult)
		fmt.Printf("Parsed rootClaim from bytes: %s (len: %d, bytes: %x)\n", rootClaim.Hex(), len(rootClaimResult), rootClaimResult)
	} else if len(rootClaimResult) > 0 {
		// Try to unpack via ABI
		if err := faultDisputeGameABI.UnpackIntoInterface(&rootClaim, "rootClaim", rootClaimResult); err != nil {
			fmt.Printf("Failed to unpack rootClaim, got error: %v (result len: %d, data: %x)\n", err, len(rootClaimResult), rootClaimResult)
			return nil, common.Hash{}, nil, fmt.Errorf("failed to unpack root claim: %w", err)
		}
	} else {
		fmt.Printf("Received empty rootClaimResult (len: %d)\n", len(rootClaimResult))
		return nil, common.Hash{}, nil, fmt.Errorf("empty root claim result from contract")
	}

	// Get the game status
	statusData, err := faultDisputeGameABI.Pack("status")
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to pack status call: %w", err)
	}

	statusResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &gameAddress,
		Data: statusData,
	}, nil)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to call status: %w", err)
	}

	var gameStatus uint8

	// First check if we got a valid result
	if len(statusResult) > 0 {
		// The status is usually just a single uint8, but might be padded to 32 bytes
		gameStatus = statusResult[len(statusResult)-1] // Take last byte
		fmt.Printf("Parsed gameStatus from bytes: %d (len: %d, bytes: %x)\n", gameStatus, len(statusResult), statusResult)
	} else {
		fmt.Printf("Received empty statusResult (len: %d)\n", len(statusResult))
		return nil, common.Hash{}, nil, fmt.Errorf("empty status result from contract")
	}

	// Step 7: Get storage proofs for the fault dispute game
	// Get proof for root claim
	rootClaimSlot := faultDisputeGameRootClaimSlot // For simplicity, using directly from config

	// Get proof for status
	statusSlot := faultDisputeGameStatusSlot // For simplicity, using directly from config

	// Get the storage proofs from L1 for the fault dispute game
	var faultDisputeGameProof types.StorageProofResult
	err = p.l1RPC.CallContext(
		ctx,
		&faultDisputeGameProof,
		"eth_getProof",
		gameAddress.Hex(),
		[]string{rootClaimSlot.Hex(), statusSlot.Hex()},
		"latest",
	)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to get fault dispute game proof: %w", err)
	}

	// Process root claim proof
	rootClaimProofIndex := -1
	for i, proof := range faultDisputeGameProof.StorageProof {
		if proof.Key == rootClaimSlot {
			rootClaimProofIndex = i
			break
		}
	}

	var faultDisputeGameRootClaimStorageProof [][]byte

	if rootClaimProofIndex == -1 {
		fmt.Printf("Root claim proof not found\n")
		return nil, common.Hash{}, nil, fmt.Errorf("root claim proof not found")
	} else {
		// Convert root claim storage proof to bytes
		faultDisputeGameRootClaimStorageProof = make([][]byte, len(faultDisputeGameProof.StorageProof[rootClaimProofIndex].Proof))
		for i, p := range faultDisputeGameProof.StorageProof[rootClaimProofIndex].Proof {
			faultDisputeGameRootClaimStorageProof[i] = common.FromHex(p)
		}
	}

	// Process status proof
	statusProofIndex := -1
	for i, proof := range faultDisputeGameProof.StorageProof {
		if proof.Key == statusSlot {
			statusProofIndex = i
			break
		}
	}

	var faultDisputeGameStatusStorageProof [][]byte

	if statusProofIndex == -1 {
		fmt.Printf("Status proof not found\n")
		return nil, common.Hash{}, nil, fmt.Errorf("status proof not found")
	} else {
		// Convert status storage proof to bytes
		faultDisputeGameStatusStorageProof = make([][]byte, len(faultDisputeGameProof.StorageProof[statusProofIndex].Proof))
		for i, p := range faultDisputeGameProof.StorageProof[statusProofIndex].Proof {
			faultDisputeGameStatusStorageProof[i] = common.FromHex(p)
		}
	}

	// Create fault dispute game state root
	faultDisputeGameStateRoot := faultDisputeGameProof.StorageHash

	// Create FaultDisputeGameStatusSlotData
	// In a real implementation, we would extract timestamp info from the contract state
	// For now, using dummy values that indicate a resolved game with valid state
	faultDisputeGameStatusData := struct {
		CreatedAt               uint64
		ResolvedAt              uint64
		GameStatus              uint8
		Initialized             bool
		L2BlockNumberChallenged bool
	}{
		CreatedAt:               12345,
		ResolvedAt:              67890,
		GameStatus:              gameStatus,
		Initialized:             true,
		L2BlockNumberChallenged: false,
	}

	rlpEncodedStatusData, err := rlp.EncodeToBytes(faultDisputeGameStatusData)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode status data: %w", err)
	}

	// Create RLP encoded fault dispute game account
	faultDisputeGameAccount := Account{
		Nonce:    uint64(*faultDisputeGameProof.Nonce),
		Balance:  faultDisputeGameProof.Balance.ToInt(),
		Root:     faultDisputeGameProof.StorageHash,
		CodeHash: faultDisputeGameProof.CodeHash.Bytes(),
	}

	rlpEncodedFaultDisputeGameData, err := rlp.EncodeToBytes(faultDisputeGameAccount)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode fault dispute game account: %w", err)
	}

	// Convert fault dispute game account proof to bytes
	faultDisputeGameAccountProof := make([][]byte, len(faultDisputeGameProof.AccountProof))
	for i, p := range faultDisputeGameProof.AccountProof {
		faultDisputeGameAccountProof[i] = common.FromHex(p)
	}

	// Step 8: Package everything together
	// Format for OPStackCannonSettledStateProof:
	// [
	//   DisputeGameFactoryProofData,
	//   FaultDisputeGameProofData
	// ]

	// DisputeGameFactoryProofData:
	disputeGameFactoryProofData := []interface{}{
		messagePasserRoot,
		latestBlockHash,
		gameIndex.Uint64(),
		crypto.Keccak256Hash(gameAddress.Bytes()), // Game ID derivation (simplified)
		disputeFaultGameStorageProof,
		rlpEncodedDisputeGameFactoryData,
		disputeGameFactoryAccountProof,
	}

	// FaultDisputeGameProofData:
	faultDisputeGameProofData := []interface{}{
		faultDisputeGameStateRoot,
		faultDisputeGameRootClaimStorageProof,
		rlpEncodedStatusData,
		faultDisputeGameStatusStorageProof,
		rlpEncodedFaultDisputeGameData,
		faultDisputeGameAccountProof,
	}

	// Combine them
	settledStateProofData := []interface{}{
		disputeGameFactoryProofData,
		faultDisputeGameProofData,
	}

	// RLP encode the final proof
	settledStateProof, err := rlp.EncodeToBytes(settledStateProofData)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to RLP encode settled state proof: %w", err)
	}

	return settledStateProof, l2StateRoot, rlpEncodedL2Header, nil
}
