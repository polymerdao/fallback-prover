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
	GetL2ConfigurationFunc          func(ctx context.Context, chainID uint64) (*t.L2ConfigInfo, error)
	GetL1BlockHashOracleFunc        func(ctx context.Context, chainID uint64) (common.Address, error)
	GetL2ConfigurationForUpdateFunc func(ctx context.Context, chainID uint64) (*t.L2Configuration, error)
	GetRegistryStorageProofFunc     func(ctx context.Context, chainID uint64) ([][]byte, []byte, [][]byte, error)
	GenerateUpdateL2ConfigArgsFunc  func(ctx context.Context, chainID uint64) (*t.UpdateL2ConfigArgs, error)
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

func (m *MockRegistryProver) GetL2ConfigurationForUpdate(
	ctx context.Context,
	chainID uint64,
) (*t.L2Configuration, error) {
	if m.GetL2ConfigurationForUpdateFunc != nil {
		return m.GetL2ConfigurationForUpdateFunc(ctx, chainID)
	}
	return nil, nil
}

func (m *MockRegistryProver) GetRegistryStorageProof(
	ctx context.Context,
	chainID uint64,
) ([][]byte, []byte, [][]byte, error) {
	if m.GetRegistryStorageProofFunc != nil {
		return m.GetRegistryStorageProofFunc(ctx, chainID)
	}
	return nil, nil, nil, nil
}

func (m *MockRegistryProver) GenerateUpdateL2ConfigArgs(
	ctx context.Context,
	chainID uint64,
) (*t.UpdateL2ConfigArgs, error) {
	if m.GenerateUpdateL2ConfigArgsFunc != nil {
		return m.GenerateUpdateL2ConfigArgsFunc(ctx, chainID)
	}
	return nil, nil
}

// MockL1OriginProver is a mock implementation of the provers.IL1OriginProver interface
type MockL1OriginProver struct {
	GetL1OriginHashFunc func(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error)
	GetL1OriginFunc     func(ctx context.Context, l1OriginHash common.Hash) ([]byte, *types.Header, error)
}

func (m *MockL1OriginProver) GetL1OriginHash(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error) {
	if m.GetL1OriginHashFunc != nil {
		return m.GetL1OriginHashFunc(ctx, l1OracleAddress)
	}
	return common.Hash{}, nil
}

func (m *MockL1OriginProver) GetL1Origin(ctx context.Context, l1Hash common.Hash) ([]byte, *types.Header, error) {
	if m.GetL1OriginFunc != nil {
		return m.GetL1OriginFunc(ctx, l1Hash)
	}
	return nil, nil, nil
}

// MockStorageProver is a mock implementation of the provers.IStorageProver interface
type MockStorageProver struct {
	GetStorageAtFunc         func(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (string, error)
	GetStorageProofFunc      func(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (*t.StorageProofResult, error)
	GenerateStorageProofFunc func(ctx context.Context, contractAddr common.Address, storageSlot common.Hash, blockNumber *big.Int) ([][]byte, []byte, [][]byte, error)
}

func (m *MockStorageProver) GetStorageAt(
	ctx context.Context,
	address common.Address,
	slot common.Hash,
	blockNumber *big.Int,
) (string, error) {
	if m.GetStorageAtFunc != nil {
		return m.GetStorageAtFunc(ctx, address, slot, blockNumber)
	}
	return "", nil
}

func (m *MockStorageProver) GetStorageProof(
	ctx context.Context,
	address common.Address,
	slot common.Hash,
	blockNumber *big.Int,
) (*t.StorageProofResult, error) {
	if m.GetStorageProofFunc != nil {
		return m.GetStorageProofFunc(ctx, address, slot, blockNumber)
	}
	return nil, nil
}

func (m *MockStorageProver) GenerateStorageProof(
	ctx context.Context,
	contractAddr common.Address,
	storageSlot common.Hash,
	blockNumber *big.Int,
) ([][]byte, []byte, [][]byte, error) {
	if m.GenerateStorageProofFunc != nil {
		return m.GenerateStorageProofFunc(ctx, contractAddr, storageSlot, big.NewInt(3))
	}
	return nil, nil, nil, nil
}

// MockOPStackBedrockProver is a mock implementation of the provers.ISettledStateProver interface
type MockOPStackBedrockProver struct {
	FindLatestResolvedFunc        func(ctx context.Context, config *t.L2ConfigInfo) (*big.Int, common.Address, error)
	GenerateSettledStateProofFunc func(ctx context.Context, l1BlockNumber, outputIndex *big.Int, rootAddress common.Address, config *t.L2ConfigInfo) ([]byte, *types.Header, error)
}

func (m *MockOPStackBedrockProver) FindLatestResolved(
	ctx context.Context,
	config *t.L2ConfigInfo,
) (*big.Int, common.Address, error) {
	if m.FindLatestResolvedFunc != nil {
		return m.FindLatestResolvedFunc(ctx, config)
	}
	return big.NewInt(0), common.Address{}, nil
}

func (m *MockOPStackBedrockProver) GenerateSettledStateProof(
	ctx context.Context,
	l1BlockNumber, outputIndex *big.Int,
	rootAddress common.Address,
	config *t.L2ConfigInfo,
) ([]byte, *types.Header, error) {
	if m.GenerateSettledStateProofFunc != nil {
		return m.GenerateSettledStateProofFunc(ctx, l1BlockNumber, outputIndex, rootAddress, config)
	}
	return nil, nil, nil
}

// MockOPStackCannonProver is a mock implementation of the provers.ISettledStateProver interface
type MockOPStackCannonProver struct {
	FindLatestResolvedFunc        func(ctx context.Context, config *t.L2ConfigInfo) (*big.Int, common.Address, error)
	GenerateSettledStateProofFunc func(ctx context.Context, l1BlockNumber, outputIndex *big.Int, rootAddress common.Address, config *t.L2ConfigInfo) ([]byte, *types.Header, error)
}

func (m *MockOPStackCannonProver) FindLatestResolved(
	ctx context.Context,
	config *t.L2ConfigInfo,
) (*big.Int, common.Address, error) {
	if m.FindLatestResolvedFunc != nil {
		return m.FindLatestResolvedFunc(ctx, config)
	}
	return big.NewInt(0), common.Address{}, nil
}

func (m *MockOPStackCannonProver) GenerateSettledStateProof(
	ctx context.Context,
	l1BlockNumber, outputIndex *big.Int,
	rootAddress common.Address,
	config *t.L2ConfigInfo,
) ([]byte, *types.Header, error) {
	if m.GenerateSettledStateProofFunc != nil {
		return m.GenerateSettledStateProofFunc(ctx, l1BlockNumber, outputIndex, rootAddress, config)
	}
	return nil, nil, nil
}
