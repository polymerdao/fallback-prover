package provers

import (
	"context"
	"fmt"
	"math/big"

	"github.com/polymerdao/fallback_prover/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// StorageProver handles generating storage proofs from L2 chains
type StorageProver struct {
	client IEthClient
	rpc    IRPCClient
}

// NewStorageProver creates a new StorageProver
func NewStorageProver(client IEthClient, rpcClient IRPCClient) *StorageProver {
	return &StorageProver{
		client: client,
		rpc:    rpcClient,
	}
}

// Account is the Ethereum account object
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash // StorageHash
	CodeHash []byte
}

// GetStorageAt gets a storage value at the given address and slot
func (s *StorageProver) GetStorageAt(
	ctx context.Context,
	address common.Address,
	slot common.Hash,
	blockNumber *big.Int,
) (string, error) {
	var result string
	err := s.rpc.CallContext(ctx, &result, "eth_getStorageAt", address.Hex(), slot.Hex(), toBlockNumArg(blockNumber))
	if err != nil {
		return "", fmt.Errorf("failed to get storage at address %s slot %s: %w", address.Hex(), slot.Hex(), err)
	}
	return result, nil
}

// GetStorageProof retrieves a storage proof for a contract address and storage slot
func (s *StorageProver) GetStorageProof(
	ctx context.Context,
	address common.Address,
	slot common.Hash,
	blockNumber *big.Int,
) (*types.StorageProofResult, error) {
	var result types.StorageProofResult

	// Use the eth_getProof RPC method to get the storage proof
	err := s.rpc.CallContext(ctx, &result, "eth_getProof", address, []string{slot.Hex()}, toBlockNumArg(blockNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to get storage proof: %w", err)
	}

	return &result, nil
}

// GenerateStorageProof creates a storage proof for the given contract and slot
func (s *StorageProver) GenerateStorageProof(
	ctx context.Context,
	contractAddr common.Address,
	storageSlot common.Hash,
	blockNumber *big.Int,
) ([][]byte, []byte, [][]byte, error) {
	// Get the storage proof from the L2 node
	proof, err := s.GetStorageProof(ctx, contractAddr, storageSlot, blockNumber) // Use latest block
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get storage proof: %w", err)
	}

	// Convert account proof to bytes
	accountProof := make([][]byte, len(proof.AccountProof))
	for i, p := range proof.AccountProof {
		accountProof[i] = common.FromHex(p)
	}

	// Get storage proof for the slot
	if len(proof.StorageProof) == 0 {
		return nil, nil, nil, fmt.Errorf("no storage proof found for slot %s", storageSlot.Hex())
	}

	// Convert storage proof to bytes
	storageProof := make([][]byte, len(proof.StorageProof[0].Proof))
	for i, p := range proof.StorageProof[0].Proof {
		storageProof[i] = common.FromHex(p)
	}

	// Create an Account object using the data from the proof
	account := Account{
		Nonce:    uint64(*proof.Nonce),
		Balance:  proof.Balance.ToInt(),
		Root:     proof.StorageHash,
		CodeHash: proof.CodeHash.Bytes(),
	}

	// RLP encode the account object
	rlpEncodedContractAccount, err := rlp.EncodeToBytes(account)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to RLP encode account: %w", err)
	}

	return storageProof, rlpEncodedContractAccount, accountProof, nil
}

// Helper function to convert big.Int block number to hex string
func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}
