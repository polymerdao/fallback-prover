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

	types2 "github.com/ethereum/go-ethereum/core/types"

	"github.com/polymerdao/fallback_prover/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
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

// constructGameID creates a GameID following the format in the Optimism contracts
// A GameID is a 32-byte identifier that combines:
// - Game type (4 bytes)
// - Creation timestamp (8 bytes)
// - Game contract address (20 bytes)
func constructGameID(gameType uint32, timestamp uint64, gameAddress common.Address) common.Hash {
	var gameID common.Hash

	// GameID structure (32 bytes total):
	// Bytes 0-3:   Game type (uint32)
	// Bytes 4-11:  Creation timestamp (uint64)
	// Bytes 12-31: Game contract address (20 bytes)

	// Convert game type to bytes and copy to the first 4 bytes
	gameTypeBytes := make([]byte, 4)
	gameTypeBytes[0] = byte(gameType >> 24) // Most significant byte
	gameTypeBytes[1] = byte(gameType >> 16)
	gameTypeBytes[2] = byte(gameType >> 8)
	gameTypeBytes[3] = byte(gameType) // Least significant byte
	copy(gameID[0:4], gameTypeBytes)

	// Convert timestamp to bytes and copy to the next 8 bytes
	timestampBytes := make([]byte, 8)
	timestampBytes[0] = byte(timestamp >> 56) // Most significant byte
	timestampBytes[1] = byte(timestamp >> 48)
	timestampBytes[2] = byte(timestamp >> 40)
	timestampBytes[3] = byte(timestamp >> 32)
	timestampBytes[4] = byte(timestamp >> 24)
	timestampBytes[5] = byte(timestamp >> 16)
	timestampBytes[6] = byte(timestamp >> 8)
	timestampBytes[7] = byte(timestamp) // Least significant byte
	copy(gameID[4:12], timestampBytes)

	// Copy the game address to the remaining 20 bytes
	copy(gameID[12:32], gameAddress.Bytes())

	return gameID
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
		},
		{
			"inputs": [],
			"name": "createdAt",
			"outputs": [
				{
					"internalType": "uint64",
					"name": "",
					"type": "uint64"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "resolvedAt",
			"outputs": [
				{
					"internalType": "uint64",
					"name": "",
					"type": "uint64"
				}
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "l2BlockNumber",
			"outputs": [
				{
					"internalType": "uint256",
					"name": "",
					"type": "uint256"
				}
			],
			"stateMutability": "pure",
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
	l1BlockNumber *big.Int,
	config *types.L2ConfigInfo) ([]byte, *types2.Header, error) {
	if len(config.Addresses) < 1 || len(config.StorageSlots) < 3 {
		return nil, nil, fmt.Errorf("invalid config: addresses or slots are insufficient")
	}

	// Get addresses and slots from the config
	disputeGameFactoryAddr := config.Addresses[0]
	disputeGameFactoryListSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[0])))
	faultDisputeGameRootClaimSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[1])))
	faultDisputeGameStatusSlot := common.BigToHash(big.NewInt(int64(config.StorageSlots[2])))

	// Interact with the DisputeGameFactory to get information about dispute games
	disputeGameFactoryABI, err := getDisputeGameFactoryABI()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse DisputeGameFactory ABI: %w", err)
	}

	// Get the total number of games
	gameCountData, err := disputeGameFactoryABI.Pack("gameCount")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack gameCount call: %w", err)
	}
	gameCountResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &disputeGameFactoryAddr,
		Data: gameCountData,
	}, l1BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call gameCount: %w", err)
	}

	// Handle the case where we receive raw bytes or properly ABI-encoded data
	var gameCount *big.Int

	// First, try to parse it as raw bytes
	if len(gameCountResult) == 32 {
		gameCount = new(big.Int).SetBytes(gameCountResult)
		log.Debug("Parsed gameCount from bytes", "count", gameCount, "len", len(gameCountResult), "bytes", fmt.Sprintf("%x", gameCountResult))
	} else if len(gameCountResult) > 0 {
		// If we have non-empty data but not 32 bytes, try ABI unpacking
		if err := disputeGameFactoryABI.UnpackIntoInterface(&gameCount, "gameCount", gameCountResult); err != nil {
			log.Debug("Failed to unpack gameCount", "error", err, "resultLen", len(gameCountResult), "data", fmt.Sprintf("%x", gameCountResult))
			return nil, nil, fmt.Errorf("failed to unpack game count: %w", err)
		}
	} else {
		log.Debug("Received empty gameCountResult", "len", len(gameCountResult))
		return nil, nil, fmt.Errorf("empty game count result from contract")
	}

	// Find the latest resolved dispute game with a valid L2 output
	var gameIndex *big.Int
	var gameAddress common.Address

	// Check game status
	faultDisputeGameABI, err := getFaultDisputeGameABI()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse FaultDisputeGame ABI: %w", err)
	}

	// Start from the most recent game and work backwards
	for i := new(big.Int).Sub(gameCount, big.NewInt(1)); i.Sign() >= 0; i.Sub(i, big.NewInt(1)) {
		// Get the game address
		gameAtIndexData, err := disputeGameFactoryABI.Pack("gameAtIndex", i)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to pack gameAtIndex call for index %v: %w", i, err)
		}

		gameAtIndexResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
			To:   &disputeGameFactoryAddr,
			Data: gameAtIndexData,
		}, l1BlockNumber)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to call gameAtIndex for index %v: %w", i, err)
		}

		var currentGameAddress common.Address
		if len(gameAtIndexResult) > 0 {
			copy(currentGameAddress[:], gameAtIndexResult[len(gameAtIndexResult)-20:]) // Take last 20 bytes
		} else {
			log.Debug("Received empty gameAtIndexResult", "index", i)
			continue
		}
		log.Debug("Game address from factory", "index", i, "address", currentGameAddress.Hex())

		statusData, err := faultDisputeGameABI.Pack("status")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to pack status call for game %s: %w", currentGameAddress.Hex(), err)
		}

		statusResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
			To:   &currentGameAddress,
			Data: statusData,
		}, l1BlockNumber)
		if err != nil {
			log.Debug("Failed to call status for game", "address", currentGameAddress.Hex(), "error", err)
			continue
		}

		var gameStatus uint8
		if len(statusResult) > 0 {
			gameStatus = statusResult[len(statusResult)-1] // Take last byte
			log.Debug("Game status", "index", i, "address", currentGameAddress.Hex(), "status", gameStatus)

			// Check if the game is resolved (status = 2 is typically RESOLVED/FINALIZED)
			// The exact status enum may vary, so adjust as needed
			if gameStatus == 2 { // Assuming 2 is RESOLVED status
				gameIndex = new(big.Int).Set(i)
				gameAddress = currentGameAddress
				break
			}
		}
	}

	if gameIndex == nil || gameIndex.Sign() < 0 {
		return nil, nil, fmt.Errorf("no suitable resolved dispute games found")
	}

	log.Debug("Using game", "index", gameIndex, "address", gameAddress.Hex())

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
		toBlockNumArg(l1BlockNumber), // Use the provided L1 block number
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get dispute game factory proof: %w", err)
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
		return nil, nil, fmt.Errorf("failed to RLP encode dispute game factory account: %w", err)
	}

	// Convert account proof to bytes
	disputeGameFactoryAccountProof := make([][]byte, len(disputeGameFactoryProof.AccountProof))
	for i, p := range disputeGameFactoryProof.AccountProof {
		disputeGameFactoryAccountProof[i] = common.FromHex(p)
	}

	// Get the root claim
	rootClaimData, err := faultDisputeGameABI.Pack("rootClaim")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack rootClaim call: %w", err)
	}

	rootClaimResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &gameAddress,
		Data: rootClaimData,
	}, l1BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call rootClaim: %w", err)
	}

	var rootClaim common.Hash

	// First check if we got a valid result
	if len(rootClaimResult) == 32 {
		copy(rootClaim[:], rootClaimResult)
		log.Debug("Parsed rootClaim from bytes", "claim", rootClaim.Hex(), "len", len(rootClaimResult), "bytes", fmt.Sprintf("%x", rootClaimResult))
	} else if len(rootClaimResult) > 0 {
		// Try to unpack via ABI
		if err := faultDisputeGameABI.UnpackIntoInterface(&rootClaim, "rootClaim", rootClaimResult); err != nil {
			log.Debug("Failed to unpack rootClaim", "error", err, "resultLen", len(rootClaimResult), "data", fmt.Sprintf("%x", rootClaimResult))
			return nil, nil, fmt.Errorf("failed to unpack root claim: %w", err)
		}
	} else {
		log.Debug("Received empty rootClaimResult", "len", len(rootClaimResult))
		return nil, nil, fmt.Errorf("empty root claim result from contract")
	}

	// Get the l2BlockNumber
	blockNumberData, err := faultDisputeGameABI.Pack("l2BlockNumber")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack l2BlockNumber call: %w", err)
	}

	l2BlockNumberResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &gameAddress,
		Data: blockNumberData,
	}, l1BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call l2BlockNumber: %w", err)
	}

	// Decode the L2 block number result
	var l2BlockNumber *big.Int
	if len(l2BlockNumberResult) == 32 {
		// For uint256 values, we can directly convert the 32 bytes to a big.Int
		l2BlockNumber = new(big.Int).SetBytes(l2BlockNumberResult)
		log.Debug("Parsed L2 block number from bytes", "blockNumber", l2BlockNumber.Uint64(), "len", len(l2BlockNumberResult))
	} else if len(l2BlockNumberResult) > 0 {
		// Try to unpack via ABI - this handles ABI-encoded data
		if err := faultDisputeGameABI.UnpackIntoInterface(&l2BlockNumber, "l2BlockNumber", l2BlockNumberResult); err != nil {
			log.Debug("Failed to unpack L2 block number", "error", err, "resultLen", len(l2BlockNumberResult), "data", fmt.Sprintf("%x", l2BlockNumberResult))
			return nil, nil, fmt.Errorf("failed to unpack L2 block number: %w", err)
		}
	} else {
		log.Debug("Received empty L2 block number result", "len", len(l2BlockNumberResult))
		return nil, nil, fmt.Errorf("empty L2 block number result from contract")
	}

	// Get the storage proofs from L1 for the fault dispute game
	var faultDisputeGameProof types.StorageProofResult
	err = p.l1RPC.CallContext(
		ctx,
		&faultDisputeGameProof,
		"eth_getProof",
		gameAddress.Hex(),
		[]string{faultDisputeGameRootClaimSlot.Hex(), faultDisputeGameStatusSlot.Hex()},
		toBlockNumArg(l1BlockNumber),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get fault dispute game proof: %w", err)
	}

	// Process root claim proof
	rootClaimProofIndex := -1
	for i, proof := range faultDisputeGameProof.StorageProof {
		if proof.Key == faultDisputeGameRootClaimSlot {
			rootClaimProofIndex = i
			break
		}
	}

	var faultDisputeGameRootClaimStorageProof [][]byte

	if rootClaimProofIndex == -1 {
		log.Debug("Root claim proof not found")
		return nil, nil, fmt.Errorf("root claim proof not found")
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
		if proof.Key == faultDisputeGameStatusSlot {
			statusProofIndex = i
			break
		}
	}

	var faultDisputeGameStatusStorageProof [][]byte

	if statusProofIndex == -1 {
		log.Debug("Status proof not found")
		return nil, nil, fmt.Errorf("status proof not found")
	} else {
		// Convert status storage proof to bytes
		faultDisputeGameStatusStorageProof = make([][]byte, len(faultDisputeGameProof.StorageProof[statusProofIndex].Proof))
		for i, p := range faultDisputeGameProof.StorageProof[statusProofIndex].Proof {
			faultDisputeGameStatusStorageProof[i] = common.FromHex(p)
		}
	}

	// Create fault dispute game state root
	faultDisputeGameStateRoot := faultDisputeGameProof.StorageHash

	// Extract real timestamps and state from the fault dispute game contract
	// Get createdAt timestamp
	createdAtData, err := faultDisputeGameABI.Pack("createdAt")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack createdAt call: %w", err)
	}

	createdAtResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &gameAddress,
		Data: createdAtData,
	}, l1BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call createdAt: %w", err)
	}

	var createdAt uint64
	if len(createdAtResult) >= 8 {
		// Parse as uint64 from the last 8 bytes
		createdAt = new(big.Int).SetBytes(createdAtResult[len(createdAtResult)-8:]).Uint64()
		log.Debug("Parsed createdAt", "timestamp", createdAt)
	} else {
		return nil, nil, fmt.Errorf("invalid createdAt result")
	}

	// Get resolvedAt timestamp
	resolvedAtData, err := faultDisputeGameABI.Pack("resolvedAt")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pack resolvedAt call: %w", err)
	}

	resolvedAtResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &gameAddress,
		Data: resolvedAtData,
	}, l1BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call resolvedAt: %w", err)
	}

	var resolvedAt uint64
	if len(resolvedAtResult) >= 8 {
		// Parse as uint64 from the last 8 bytes
		resolvedAt = new(big.Int).SetBytes(resolvedAtResult[len(resolvedAtResult)-8:]).Uint64()
		log.Debug("Parsed resolvedAt", "timestamp", resolvedAt)
	} else {
		return nil, nil, fmt.Errorf("invalid resolvedAt result")
	}

	// Create the status data structure with real values from the contract
	faultDisputeGameStatusData := struct {
		CreatedAt               uint64
		ResolvedAt              uint64
		GameStatus              uint8
		Initialized             bool
		L2BlockNumberChallenged bool
	}{
		CreatedAt:               createdAt,
		ResolvedAt:              resolvedAt,
		GameStatus:              2,
		Initialized:             true, // must be true if the game resolved
		L2BlockNumberChallenged: true, // must be true if the game resolved in favor of defender
	}

	rlpEncodedStatusData, err := rlp.EncodeToBytes(faultDisputeGameStatusData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to RLP encode status data: %w", err)
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
		return nil, nil, fmt.Errorf("failed to RLP encode fault dispute game account: %w", err)
	}

	// Convert fault dispute game account proof to bytes
	faultDisputeGameAccountProof := make([][]byte, len(faultDisputeGameProof.AccountProof))
	for i, p := range faultDisputeGameProof.AccountProof {
		faultDisputeGameAccountProof[i] = common.FromHex(p)
	}

	// Get the messagePasserRoot corresponding to this settled L2 state
	messagePasserAddr := common.HexToAddress(CannonL2MessagePasserAddress)

	// Get the storage root (storageHash) of the L2ToL1MessagePasser contract
	var messagePasserProof types.StorageProofResult
	err = p.l2RPC.CallContext(
		ctx,
		&messagePasserProof,
		"eth_getProof",
		messagePasserAddr.Hex(),
		[]string{}, // No specific storage keys needed, we only want account info
		toBlockNumArg(l2BlockNumber),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get message passer proof: %w", err)
	}
	messagePasserRoot := messagePasserProof.StorageHash

	// Get the blockhash for this height
	l2Block, err := p.l2Client.BlockByNumber(ctx, l2BlockNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get L2 block header: %w", err)
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
		l2Block.Hash(),
		gameIndex.Uint64(),
		constructGameID(0, createdAt, gameAddress), // Construct proper GameID with type 0 (fault), creation timestamp, and address
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
		return nil, nil, fmt.Errorf("failed to RLP encode settled state proof: %w", err)
	}

	return settledStateProof, l2Block.Header(), nil
}
