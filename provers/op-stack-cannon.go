package provers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"

	types2 "github.com/ethereum/go-ethereum/core/types"

	"github.com/polymerdao/fallback_prover/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var _ ISettledStateProver = &OPStackCannonProver{}

// OPStackCannonProver handles proof generation for OP Stack Cannon chains
type OPStackCannonProver struct {
	l1Client   IEthClient
	l1RPC      IRPCClient
	l2RPC      IRPCClient
	abi        abi.ABI
	factoryABI abi.ABI
	gameABI    abi.ABI
}

// NewOPStackCannonProver creates a new prover instance for OP Stack Cannon
func NewOPStackCannonProver(l1Client IEthClient, l1RPC, l2RPC IRPCClient) (*OPStackCannonProver, error) {
	abiObj, err := getOPStackCannonProverABI()
	if err != nil {
		return nil, err
	}

	factoryAbiObj, err := getDisputeGameFactoryABI()
	if err != nil {
		return nil, err
	}

	gameABIObj, err := getFaultDisputeGameABI()
	if err != nil {
		return nil, err
	}

	return &OPStackCannonProver{
		l1Client:   l1Client,
		l1RPC:      l1RPC,
		l2RPC:      l2RPC,
		abi:        abiObj,
		factoryABI: factoryAbiObj,
		gameABI:    gameABIObj,
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

func (p *OPStackCannonProver) FindLatestResolved(ctx context.Context, config *types.L2ConfigInfo) (*big.Int, common.Address, error) {
	if len(config.Addresses) < 1 || len(config.StorageSlots) < 3 {
		return nil, common.Address{}, fmt.Errorf("invalid config: addresses or slots are insufficient")
	}

	// Get addresses and slots from the config
	disputeGameFactoryAddr := config.Addresses[0]

	// Get the total number of games
	gameCountData, err := p.factoryABI.Pack("gameCount")
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to pack gameCount call: %w", err)
	}
	gameCountResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &disputeGameFactoryAddr,
		Data: gameCountData,
	}, nil)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("failed to call gameCount: %w", err)
	}

	// Handle the case where we receive raw bytes or properly ABI-encoded data
	var gameCount *big.Int

	// First, try to parse it as raw bytes
	if len(gameCountResult) == 32 {
		gameCount = new(big.Int).SetBytes(gameCountResult)
		log.Debug("Parsed gameCount from bytes", "count", gameCount, "len", len(gameCountResult), "bytes", fmt.Sprintf("%x", gameCountResult))
	} else {
		log.Debug("Received empty gameCountResult", "len", len(gameCountResult))
		return nil, common.Address{}, fmt.Errorf("empty game count result from contract")
	}
	if gameCount.Cmp(big.NewInt(0)) <= 0 {
		return nil, common.Address{}, fmt.Errorf("invalid game count %s", gameCount.String())
	}

	// Find the latest resolved dispute game with a valid L2 output
	var gameIndex *big.Int
	var gameAddress common.Address

	// Start from the most recent game and work backwards
	for i := new(big.Int).Sub(gameCount, big.NewInt(1)); i.Sign() >= 0; i.Sub(i, big.NewInt(1)) {
		// Get the game address
		gameAtIndexData, err := p.factoryABI.Pack("gameAtIndex", i)
		if err != nil {
			return nil, common.Address{}, fmt.Errorf("failed to pack gameAtIndex call for index %v: %w", i, err)
		}

		gameAtIndexResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
			To:   &disputeGameFactoryAddr,
			Data: gameAtIndexData,
		}, nil)
		if err != nil {
			return nil, common.Address{}, fmt.Errorf("failed to call gameAtIndex for index %v: %w", i, err)
		}

		var currentGameAddress common.Address
		if len(gameAtIndexResult) > 0 {
			copy(currentGameAddress[:], gameAtIndexResult[len(gameAtIndexResult)-20:]) // Take last 20 bytes
		} else {
			log.Debug("Received empty gameAtIndexResult", "index", i)
			continue
		}
		log.Debug("Game address from factory", "index", i, "address", currentGameAddress.Hex())

		statusData, err := p.gameABI.Pack("status")
		if err != nil {
			return nil, common.Address{}, fmt.Errorf("failed to pack status call for game %s: %w", currentGameAddress.Hex(), err)
		}

		statusResult, err := p.l1Client.CallContract(ctx, ethereum.CallMsg{
			To:   &currentGameAddress,
			Data: statusData,
		}, nil)
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
		return nil, common.Address{}, fmt.Errorf("no suitable resolved dispute games found")
	}
	return gameIndex, gameAddress, nil
}

// GenerateSettledStateProof creates a proof for an OPStack Cannon L2 against L1
func (p *OPStackCannonProver) GenerateSettledStateProof(
	ctx context.Context,
	l1BlockNumber, gameIndex *big.Int,
	gameAddress common.Address,
	config *types.L2ConfigInfo) ([]byte, *types2.Header, error) {
	if len(config.Addresses) < 1 || len(config.StorageSlots) < 3 {
		return nil, nil, fmt.Errorf("invalid config: addresses or slots are insufficient")
	}

	// Get addresses and slots from the config
	disputeGameFactoryAddr := config.Addresses[0]
	disputeGameFactoryListSlot := common.BigToHash(new(big.Int).SetUint64(config.StorageSlots[0]))
	faultDisputeGameRootClaimSlot := common.BigToHash(new(big.Int).SetUint64(config.StorageSlots[1]))
	faultDisputeGameStatusSlot := common.BigToHash(new(big.Int).SetUint64(config.StorageSlots[2]))

	log.Debug("Using game", "index", gameIndex, "address", gameAddress.Hex())

	// Get storage proof for the dispute game factory
	// Calculate the storage slot for the game index
	gameIndexSlot := crypto.Keccak256Hash(
		common.LeftPadBytes(gameIndex.Bytes(), 32),
		common.LeftPadBytes(disputeGameFactoryListSlot.Bytes(), 32),
	)

	var rawFactoryProof json.RawMessage
	factoryProofElem := rpc.BatchElem{
		Method: "eth_getProof",
		Args: []interface{}{
			disputeGameFactoryAddr.Hex(),
			[]string{gameIndexSlot.Hex()},
			toBlockNumArg(l1BlockNumber),
		},
		Result: &rawFactoryProof,
		Error:  nil,
	}

	// Get the storage proofs from L1 for the fault dispute game
	var rawGameProof json.RawMessage
	gameProofElem := rpc.BatchElem{
		Method: "eth_getProof",
		Args: []interface{}{
			gameAddress.Hex(),
			[]string{faultDisputeGameRootClaimSlot.Hex(), faultDisputeGameStatusSlot.Hex()},
			toBlockNumArg(l1BlockNumber),
		},
		Result: &rawGameProof,
		Error:  nil,
	}

	l1BatchElems := []rpc.BatchElem{factoryProofElem, gameProofElem}
	if err := p.l1RPC.BatchCallContext(ctx, l1BatchElems); err != nil {
		return nil, nil, fmt.Errorf("failed to batch call: %w", err)
	}
	for _, elem := range l1BatchElems {
		if elem.Error != nil {
			return nil, nil, fmt.Errorf("l1 RPC batch request error for method %s: %w", elem.Method, elem.Error)
		}
		if elem.Result == nil {
			return nil, nil, fmt.Errorf("l1 RPC batch request result is nil for method %s", elem.Method)
		}
	}

	var faultDisputeGameProof types.StorageProofResult
	if err := json.Unmarshal(rawGameProof, &faultDisputeGameProof); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal game proof: %w", err)
	}
	var disputeGameFactoryProof types.StorageProofResult
	if err := json.Unmarshal(rawFactoryProof, &disputeGameFactoryProof); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal factory proof: %w", err)
	}

	// Get the l2BlockNumber
	blockNumberData, err := p.gameABI.Pack("l2BlockNumber")
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

	// Get createdAt timestamp
	createdAtData, err := p.gameABI.Pack("createdAt")
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

	// Get resolvedAt timestamp
	resolvedAtData, err := p.gameABI.Pack("resolvedAt")
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

	// Convert storage proof to bytes
	disputeFaultGameStorageProof, rlpEncodedDisputeGameFactoryData, disputeGameFactoryAccountProof, err := processAccountAndProofs(&disputeGameFactoryProof)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process account and proofs: %w", err)
	}

	// Decode the L2 block number result
	var l2BlockNumber *big.Int
	if len(l2BlockNumberResult) == 32 {
		// For uint256 values, we can directly convert the 32 bytes to a big.Int
		l2BlockNumber = new(big.Int).SetBytes(l2BlockNumberResult)
		log.Debug("Parsed L2 block number from bytes", "blockNumber", l2BlockNumber.Uint64(), "len", len(l2BlockNumberResult))
	} else {
		log.Debug("Received empty L2 block number result", "len", len(l2BlockNumberResult))
		return nil, nil, fmt.Errorf("empty L2 block number result from contract")
	}
	if l2BlockNumber.Cmp(big.NewInt(0)) <= 0 {
		return nil, nil, fmt.Errorf("invalid l2 block number %s", l2BlockNumber.String())
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

	var createdAt uint64
	if len(createdAtResult) >= 8 {
		// Parse as uint64 from the last 8 bytes
		createdAt = new(big.Int).SetBytes(createdAtResult[len(createdAtResult)-8:]).Uint64()
		log.Debug("Parsed createdAt", "timestamp", createdAt)
	} else {
		return nil, nil, fmt.Errorf("invalid createdAt result")
	}
	if createdAt <= 0 {
		return nil, nil, fmt.Errorf("invalid createdAt %d", createdAt)
	}

	var resolvedAt uint64
	if len(resolvedAtResult) >= 8 {
		// Parse as uint64 from the last 8 bytes
		resolvedAt = new(big.Int).SetBytes(resolvedAtResult[len(resolvedAtResult)-8:]).Uint64()
		log.Debug("Parsed resolvedAt", "timestamp", resolvedAt)
	} else {
		return nil, nil, fmt.Errorf("invalid resolvedAt result")
	}
	if resolvedAt <= 0 {
		return nil, nil, fmt.Errorf("invalid resolvedAt %d", resolvedAt)
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
	var rawL2Proof json.RawMessage
	messagePasserProofElem := rpc.BatchElem{
		Method: "eth_getProof",
		Args: []interface{}{
			messagePasserAddr.Hex(),
			[]string{}, // No specific storage keys needed, we only want account info
			toBlockNumArg(l2BlockNumber),
		},
		Result: &rawL2Proof,
		Error:  nil,
	}

	// Get the blockhash for this height
	var rawHeader json.RawMessage
	l2BlockElem := rpc.BatchElem{
		Method: "eth_getBlockByNumber",
		Args: []interface{}{
			toBlockNumArg(l2BlockNumber),
			false,
		},
		Result: &rawHeader,
		Error:  nil,
	}

	l2BatchElems := []rpc.BatchElem{l2BlockElem, messagePasserProofElem}
	if err := p.l2RPC.BatchCallContext(ctx, l2BatchElems); err != nil {
		return nil, nil, fmt.Errorf("failed to batch call: %w", err)
	}
	for _, elem := range l2BatchElems {
		if elem.Error != nil {
			return nil, nil, fmt.Errorf("l2 RPC batch request error for method %s: %w", elem.Method, elem.Error)
		}
		if elem.Result == nil {
			return nil, nil, fmt.Errorf("l2 RPC batch request result is nil for method %s", elem.Method)
		}
	}

	var l2Header types2.Header
	if err := json.Unmarshal(rawHeader, &l2Header); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal L2 block header: %w", err)
	}

	var messagePasserProof types.StorageProofResult
	if err := json.Unmarshal(rawL2Proof, &messagePasserProof); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal message passer proof: %w", err)
	}

	messagePasserRoot := messagePasserProof.StorageHash

	// Package everything together
	// Format for OPStackCannonSettledStateProof:
	// [
	//   DisputeGameFactoryProofData,
	//   FaultDisputeGameProofData
	// ]

	// DisputeGameFactoryProofData:
	disputeGameFactoryProofData := []interface{}{
		messagePasserRoot,
		l2Header.Hash(),
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

	return settledStateProof, &l2Header, nil
}
