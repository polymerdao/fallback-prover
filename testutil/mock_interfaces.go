package testutil

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

func (m *MockRegistryProver) GetL2ConfigurationForUpdate(ctx context.Context, chainID uint64) (*t.L2Configuration, error) {
	if m.GetL2ConfigurationForUpdateFunc != nil {
		return m.GetL2ConfigurationForUpdateFunc(ctx, chainID)
	}
	return nil, nil
}

func (m *MockRegistryProver) GetRegistryStorageProof(ctx context.Context, chainID uint64) ([][]byte, []byte, [][]byte, error) {
	if m.GetRegistryStorageProofFunc != nil {
		return m.GetRegistryStorageProofFunc(ctx, chainID)
	}
	return nil, nil, nil, nil
}

func (m *MockRegistryProver) GenerateUpdateL2ConfigArgs(ctx context.Context, chainID uint64) (*t.UpdateL2ConfigArgs, error) {
	if m.GenerateUpdateL2ConfigArgsFunc != nil {
		return m.GenerateUpdateL2ConfigArgsFunc(ctx, chainID)
	}
	return nil, nil
}

// MockL1OriginProver is a mock implementation of the provers.IL1OriginProver interface
type MockL1OriginProver struct {
	ProveL1OriginFunc func(ctx context.Context, l1OracleAddress common.Address) ([]byte, *types.Header, error)
}

func (m *MockL1OriginProver) ProveL1Origin(ctx context.Context, l1OracleAddress common.Address) ([]byte, *types.Header, error) {
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
	GenerateSettledStateProofFunc func(ctx context.Context, l1BlockNumber *big.Int, config *t.L2ConfigInfo) ([]byte, *types.Header, error)
}

func (m *MockOPStackBedrockProver) GenerateSettledStateProof(ctx context.Context, l1BlockNumber *big.Int, config *t.L2ConfigInfo) ([]byte, *types.Header, error) {
	if m.GenerateSettledStateProofFunc != nil {
		return m.GenerateSettledStateProofFunc(ctx, l1BlockNumber, config)
	}
	return nil, nil, nil
}

// MockOPStackCannonProver is a mock implementation of the provers.ISettledStateProver interface
type MockOPStackCannonProver struct {
	GenerateSettledStateProofFunc func(ctx context.Context, l1BlockNumber *big.Int, config *t.L2ConfigInfo) ([]byte, *types.Header, error)
}

func (m *MockOPStackCannonProver) GenerateSettledStateProof(ctx context.Context, l1BlockNumber *big.Int, config *t.L2ConfigInfo) ([]byte, *types.Header, error) {
	if m.GenerateSettledStateProofFunc != nil {
		return m.GenerateSettledStateProofFunc(ctx, l1BlockNumber, config)
	}
	return nil, nil, nil
}

// MockNativeProver is a mock implementation of the provers.INativeProver interface
type MockNativeProver struct {
	// The implementation still uses the older style for backward compatibility with tests
	EncodeProveCalldataFunc             func(chainID *big.Int, contractAddr common.Address, storageSlot common.Hash, storageValue common.Hash, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, l2WorldStateRoot common.Hash, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error)
	EncodeUpdateAndProveCalldataFunc    func(updateArgs t.UpdateL2ConfigArgs, proveArgs t.ProveScalarArgs, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error)
	EncodeConfigureAndProveCalldataFunc func(updateArgs t.UpdateL2ConfigArgs, proveArgs t.ProveScalarArgs, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error)
	EncodeProveL1CalldataFunc           func(proveArgs t.ProveL1ScalarArgs, rlpEncodedL1Header []byte, l1StorageProof [][]byte, rlpEncodedContractAccount []byte, l1AccountProof [][]byte) ([]byte, error)
	GetABIFunc                          func() abi.ABI
}

func (m *MockNativeProver) EncodeProveCalldata(
	proveArgs t.ProveScalarArgs,
	rlpEncodedL1Header []byte,
	rlpEncodedL2Header []byte,
	settledStateProof []byte,
	l2StorageProof [][]byte,
	rlpEncodedContractAccount []byte,
	l2AccountProof [][]byte,
) ([]byte, error) {
	if m.EncodeProveCalldataFunc != nil {
		return m.EncodeProveCalldataFunc(proveArgs.ChainID, proveArgs.ContractAddr, proveArgs.StorageSlot, proveArgs.StorageValue, rlpEncodedL1Header, rlpEncodedL2Header, proveArgs.L2WorldStateRoot, settledStateProof, l2StorageProof, rlpEncodedContractAccount, l2AccountProof)
	}
	return nil, nil
}

func (m *MockNativeProver) EncodeUpdateAndProveCalldata(updateArgs t.UpdateL2ConfigArgs, proveArgs t.ProveScalarArgs, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error) {
	if m.EncodeUpdateAndProveCalldataFunc != nil {
		return m.EncodeUpdateAndProveCalldataFunc(updateArgs, proveArgs, rlpEncodedL1Header, rlpEncodedL2Header, settledStateProof, l2StorageProof, rlpEncodedContractAccount, l2AccountProof)
	}
	return nil, nil
}

func (m *MockNativeProver) EncodeConfigureAndProveCalldata(updateArgs t.UpdateL2ConfigArgs, proveArgs t.ProveScalarArgs, rlpEncodedL1Header []byte, rlpEncodedL2Header []byte, settledStateProof []byte, l2StorageProof [][]byte, rlpEncodedContractAccount []byte, l2AccountProof [][]byte) ([]byte, error) {
	if m.EncodeConfigureAndProveCalldataFunc != nil {
		return m.EncodeConfigureAndProveCalldataFunc(updateArgs, proveArgs, rlpEncodedL1Header, rlpEncodedL2Header, settledStateProof, l2StorageProof, rlpEncodedContractAccount, l2AccountProof)
	}
	return nil, nil
}

func (m *MockNativeProver) EncodeProveL1Calldata(
	proveArgs t.ProveL1ScalarArgs,
	rlpEncodedL1Header []byte,
	l1StorageProof [][]byte,
	rlpEncodedContractAccount []byte,
	l1AccountProof [][]byte,
) ([]byte, error) {
	if m.EncodeProveL1CalldataFunc != nil {
		return m.EncodeProveL1CalldataFunc(
			proveArgs,
			rlpEncodedL1Header,
			l1StorageProof,
			rlpEncodedContractAccount,
			l1AccountProof,
		)
	}
	return nil, nil
}

func (m *MockNativeProver) GetABI() abi.ABI {
	if m.GetABIFunc != nil {
		return m.GetABIFunc()
	}
	return abi.ABI{}
}
