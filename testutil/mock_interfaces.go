package testutil

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	t "github.com/polymerdao/fallback_prover/types"
)

// MockRegistryProver is a mock implementation of the provers.IRegistryProver interface
type MockRegistryProver struct {
	GetL2ConfigurationFunc   func(ctx context.Context, chainID uint64) (*t.L2ConfigInfo, error)
	GetL1BlockHashOracleFunc func(ctx context.Context, chainID uint64) (common.Address, error)
}

func (m *MockRegistryProver) GetL2Configuration(ctx context.Context, chainID uint64) (*t.L2ConfigInfo, error) {
	if m.GetL2ConfigurationFunc != nil {
		return m.GetL2ConfigurationFunc(ctx, chainID)
	}
	return nil, nil
}

func (m *MockRegistryProver) GetL1BlockHashOracle(ctx context.Context, chainID uint64) (common.Address, error) {
	if m.GetL1BlockHashOracleFunc != nil {
		return m.GetL1BlockHashOracleFunc(ctx, chainID)
	}
	return common.Address{}, nil
}

// MockL1OriginProver is a mock implementation of the provers.IL1OriginProver interface
type MockL1OriginProver struct {
	ProveL1OriginFunc func(ctx context.Context, l1OracleAddress common.Address) ([]byte, *types.Block, error)
}

func (m *MockL1OriginProver) ProveL1Origin(ctx context.Context, l1OracleAddress common.Address) ([]byte, *types.Block, error) {
	if m.ProveL1OriginFunc != nil {
		return m.ProveL1OriginFunc(ctx, l1OracleAddress)
	}
	return nil, nil, nil
}

// MockStorageProver is a mock implementation of the provers.IStorageProver interface
type MockStorageProver struct {
	GetStorageAtFunc         func(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (string, error)
	GetStorageProofFunc      func(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (*t.StorageProofResult, error)
	GenerateStorageProofFunc func(ctx context.Context, contractAddr common.Address, storageSlot common.Hash, stateRoot common.Hash) ([][]byte, []byte, [][]byte, error)
}

func (m *MockStorageProver) GetStorageAt(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (string, error) {
	if m.GetStorageAtFunc != nil {
		return m.GetStorageAtFunc(ctx, address, slot, blockNumber)
	}
	return "", nil
}

func (m *MockStorageProver) GetStorageProof(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (*t.StorageProofResult, error) {
	if m.GetStorageProofFunc != nil {
		return m.GetStorageProofFunc(ctx, address, slot, blockNumber)
	}
	return nil, nil
}

func (m *MockStorageProver) GenerateStorageProof(ctx context.Context, contractAddr common.Address, storageSlot common.Hash, stateRoot common.Hash) ([][]byte, []byte, [][]byte, error) {
	if m.GenerateStorageProofFunc != nil {
		return m.GenerateStorageProofFunc(ctx, contractAddr, storageSlot, stateRoot)
	}
	return nil, nil, nil, nil
}

// MockOPStackBedrockProver is a mock implementation of the provers.ISettledStateProver interface
type MockOPStackBedrockProver struct {
	GenerateSettledStateProofFunc func(ctx context.Context, config *t.L2ConfigInfo, l1StateRoot common.Hash) ([]byte, common.Hash, []byte, error)
}

func (m *MockOPStackBedrockProver) GenerateSettledStateProof(ctx context.Context, config *t.L2ConfigInfo, l1StateRoot common.Hash) ([]byte, common.Hash, []byte, error) {
	if m.GenerateSettledStateProofFunc != nil {
		return m.GenerateSettledStateProofFunc(ctx, config, l1StateRoot)
	}
	return nil, common.Hash{}, nil, nil
}

// MockOPStackCannonProver is a mock implementation of the provers.ISettledStateProver interface
type MockOPStackCannonProver struct {
	GenerateSettledStateProofFunc func(ctx context.Context, config *t.L2ConfigInfo, l1StateRoot common.Hash) ([]byte, common.Hash, []byte, error)
}

func (m *MockOPStackCannonProver) GenerateSettledStateProof(ctx context.Context, config *t.L2ConfigInfo, l1StateRoot common.Hash) ([]byte, common.Hash, []byte, error) {
	if m.GenerateSettledStateProofFunc != nil {
		return m.GenerateSettledStateProofFunc(ctx, config, l1StateRoot)
	}
	return nil, common.Hash{}, nil, nil
}

// MockNativeProver is a mock implementation of the provers.INativeProver interface
type MockNativeProver struct {
	EncodeProveCalldataFunc func(chainID *big.Int, contractAddr common.Address, storageSlot common.Hash, storageValue common.Hash, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, l2WorldStateRoot common.Hash, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error)
}

func (m *MockNativeProver) EncodeProveCalldata(chainID *big.Int, contractAddr common.Address, storageSlot common.Hash, storageValue common.Hash, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, l2WorldStateRoot common.Hash, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error) {
	if m.EncodeProveCalldataFunc != nil {
		return m.EncodeProveCalldataFunc(chainID, contractAddr, storageSlot, storageValue, rlpEncodedL1Header, rlpEncodedL2Header, l2WorldStateRoot, settledStateProof, l2StorageProof, rlpEncodedContractAccount, l2AccountProof)
	}
	return nil, nil
}
