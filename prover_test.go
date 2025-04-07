package fallback_prover

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/polymerdao/fallback_prover/provers"
	"github.com/polymerdao/fallback_prover/testutil"
	types2 "github.com/polymerdao/fallback_prover/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProver_GenerateProofCalldata(t *testing.T) {
	// Create test data
	srcL2ChainID := uint64(10)    // Optimism chain ID
	dstL2ChainID := uint64(42161) // Arbitrum chain ID
	srcAddress := "0x1234567890abcdef1234567890abcdef12345678"
	srcStorageSlot := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// Create a test header and block
	l1Header := testutil.CreateTestHeader(t)
	l1Block := testutil.CreateTestBlock(t, l1Header)
	l2Header := testutil.CreateTestHeader(t)

	// RLP encode headers
	rlpEncodedL1Header, err := rlp.EncodeToBytes(l1Header)
	require.NoError(t, err)
	rlpEncodedL2Header, err := rlp.EncodeToBytes(l2Header)
	require.NoError(t, err)

	// Mock settled state proof data
	mockSettledStateProof := []byte("mock-settled-state-proof")
	mockStorageProof := [][]byte{[]byte("storage-proof-1"), []byte("storage-proof-2")}
	mockEncodedContractAccount := []byte("mock-encoded-contract-account")
	mockAccountProof := [][]byte{[]byte("account-proof-1"), []byte("account-proof-2")}

	// Create a test config using our testutil.L2ConfigInfo
	testConfig := &types2.L2ConfigInfo{
		ConfigType: "OPStackBedrock",
		Addresses: []common.Address{
			common.HexToAddress("0x1234"),
		},
		StorageSlots: []uint64{0x123},
	}

	// Create mock provers
	mockRegistryProver := &testutil.MockRegistryProver{
		GetL2ConfigurationFunc: func(ctx context.Context, chainID uint64) (*types2.L2ConfigInfo, error) {
			return testConfig, nil
		},
		GetL1BlockHashOracleFunc: func(ctx context.Context, chainID uint64) (common.Address, error) {
			return common.HexToAddress("0x5678"), nil
		},
	}

	mockL1OriginProver := &testutil.MockL1OriginProver{
		ProveL1OriginFunc: func(ctx context.Context, l1OracleAddress common.Address) ([]byte, *types.Block, error) {
			return rlpEncodedL1Header, l1Block, nil
		},
	}

	mockStorageProver := &testutil.MockStorageProver{
		GetStorageAtFunc: func(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (string, error) {
			return "0x0000000000000000000000000000000000000000000000000000000000000123", nil
		},
		GenerateStorageProofFunc: func(ctx context.Context, contractAddr common.Address, storageSlot common.Hash, stateRoot common.Hash) ([][]byte, []byte, [][]byte, error) {
			return mockStorageProof, mockEncodedContractAccount, mockAccountProof, nil
		},
	}

	mockBedrockProver := &testutil.MockOPStackBedrockProver{
		GenerateSettledStateProofFunc: func(ctx context.Context, config *types2.L2ConfigInfo, l1StateRoot common.Hash) ([]byte, common.Hash, []byte, error) {
			return mockSettledStateProof, l2Header.Root, rlpEncodedL2Header, nil
		},
	}

	nativeProver, err := provers.NewNativeProver()
	require.NoError(t, err)

	// Create the Prover instance with mocked interfaces
	prover := &Prover{
		registryProver:     mockRegistryProver,
		l1OriginProver:     mockL1OriginProver,
		nativeProver:       nativeProver,
		l2StorageProver:    mockStorageProver,
		settledStateProver: mockBedrockProver,
		l2Config:           testConfig,
	}

	// Call the method being tested
	calldata, err := prover.GenerateProofCalldata(
		context.Background(),
		srcL2ChainID,
		dstL2ChainID,
		srcAddress,
		srcStorageSlot,
	)
	require.NoError(t, err)

	// Verify the result
	assert.Equal(t, "0x1234567890abcdef", calldata)
}