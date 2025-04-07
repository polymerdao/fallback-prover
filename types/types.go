package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// L2ConfigInfo contains the configuration for an L2 chain
type L2ConfigInfo struct {
	ConfigType   string
	Addresses    []common.Address
	StorageSlots []uint64
}

// StorageProofResult contains the result of a storage proof request
type StorageProofResult struct {
	Address      common.Address      `json:"address"`
	AccountProof []string            `json:"accountProof"`
	Balance      *hexutil.Big        `json:"balance"`
	CodeHash     common.Hash         `json:"codeHash"`
	Nonce        *hexutil.Uint64     `json:"nonce"`
	StorageHash  common.Hash         `json:"storageHash"`
	StorageProof []StorageProofEntry `json:"storageProof"`
}

// StorageProofEntry represents a single storage entry in a proof
type StorageProofEntry struct {
	Key   common.Hash  `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}
