package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// L2ConfigInfo contains the configuration for an L2 chain
type L2ConfigInfo struct {
	ConfigType   string
	Addresses    []common.Address
	StorageSlots []*big.Int
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

// L2Type represents the type of L2 chain
type L2Type uint8

const (
	Nitro L2Type = iota
	OPStackBedrock
	OPStackCannon
)

// L2Configuration represents the L2 chain configuration
type L2Configuration struct {
	Prover               common.Address
	Addresses            []common.Address
	StorageSlots         []*big.Int
	VersionNumber        *big.Int
	FinalityDelaySeconds *big.Int
	L2Type               L2Type
}

// UpdateL2ConfigArgs represents the arguments needed for updating an L2 configuration
type UpdateL2ConfigArgs struct {
	Config                        L2Configuration
	L1StorageProof                [][]byte
	RlpEncodedRegistryAccountData []byte
	L1RegistryProof               [][]byte
}

// ProveScalarArgs holds scalar arguments for the prove function
type ProveScalarArgs struct {
	ChainID          *big.Int
	ContractAddr     common.Address
	StorageSlot      common.Hash
	StorageValue     common.Hash
	L2WorldStateRoot common.Hash
}

// ProveL1ScalarArgs holds scalar arguments for the proveL1 function
type ProveL1ScalarArgs struct {
	ContractAddr     common.Address
	StorageSlot      common.Hash
	StorageValue     common.Hash
	L1WorldStateRoot common.Hash
}
