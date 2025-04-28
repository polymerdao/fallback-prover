package provers

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	t "github.com/polymerdao/fallback_prover/types"
)

type IEthClient interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

type IRPCClient interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type IL1OriginProver interface {
	ProveL1Origin(ctx context.Context, l1OracleAddress common.Address) ([]byte, *types.Header, error)
}

type IStorageProver interface {
	GenerateStorageProof(
		ctx context.Context,
		contractAddr common.Address,
		storageSlot common.Hash,
		stateRoot common.Hash,
	) ([][]byte, []byte, [][]byte, error)
	GetStorageProof(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (*t.StorageProofResult, error)
	GetStorageAt(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (string, error)
}

type INativeProver interface {
	EncodeProveCalldata(
		chainID *big.Int,
		contractAddr common.Address,
		storageSlot common.Hash,
		storageValue common.Hash,
		rlpEncodedL1Header []byte,
		rlpEncodedL2Header []byte,
		l2WorldStateRoot common.Hash,
		settledStateProof []byte,
		l2StorageProof [][]byte,
		rlpEncodedContractAccount []byte,
		l2AccountProof [][]byte,
	) ([]byte, error)

	// Added methods for the new contract functions
	EncodeUpdateAndProveCalldata(
		updateArgs t.UpdateL2ConfigArgs,
		proveArgs t.ProveScalarArgs,
		rlpEncodedL1Header []byte,
		rlpEncodedL2Header []byte,
		settledStateProof []byte,
		l2StorageProof [][]byte,
		rlpEncodedContractAccount []byte,
		l2AccountProof [][]byte,
	) ([]byte, error)

	EncodeConfigureAndProveCalldata(
		updateArgs t.UpdateL2ConfigArgs,
		proveArgs t.ProveScalarArgs,
		rlpEncodedL1Header []byte,
		rlpEncodedL2Header []byte,
		settledStateProof []byte,
		l2StorageProof [][]byte,
		rlpEncodedContractAccount []byte,
		l2AccountProof [][]byte,
	) ([]byte, error)

	EncodeProveL1Calldata(
		proveArgs t.ProveL1ScalarArgs,
		rlpEncodedL1Header []byte,
		l1StorageProof [][]byte,
		rlpEncodedContractAccount []byte,
		l1AccountProof [][]byte,
	) ([]byte, error)

	// For testing purposes
	GetABI() abi.ABI
}

type ISettledStateProver interface {
	GenerateSettledStateProof(
		ctx context.Context,
		config *t.L2ConfigInfo) ([]byte, common.Hash, []byte, error)
}

type IRegistryProver interface {
	GetL2Configuration(ctx context.Context, chainID uint64) (*t.L2ConfigInfo, error)
	GetL1BlockHashOracle(ctx context.Context, chainID uint64) (common.Address, error)
	GetL2ConfigurationForUpdate(ctx context.Context, chainID uint64) (*t.L2Configuration, error)
	GetRegistryStorageProof(ctx context.Context, chainID uint64) ([][]byte, []byte, [][]byte, error)
	GenerateUpdateL2ConfigArgs(ctx context.Context, chainID uint64) (*t.UpdateL2ConfigArgs, error)
}
