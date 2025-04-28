package provers

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/polymerdao/fallback_prover/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL1OriginProver_ProveL1Origin(t *testing.T) {
	// Create test data
	l1OracleAddress := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	l1HeaderHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	// Create a test header and block
	header := testutil.CreateTestHeader(t)
	block := testutil.CreateTestBlock(t, header)

	// Expected RLP encoded header
	expectedEncodedHeader, err := rlp.EncodeToBytes(header)
	require.NoError(t, err)

	// Create mock L2 client
	mockL2Client := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the L1 oracle contract
			testutil.RequireAddressEq(t, l1OracleAddress, *msg.To)

			// Return the hash - the function only uses 32 bytes, so we construct a result that contains that
			return l1HeaderHash.Bytes(), nil
		},
	}

	// Create mock L1 client
	mockL1Client := &testutil.MockEthClient{
		BlockByHashFunc: func(ctx context.Context, hash common.Hash) (*types.Block, error) {
			assert.Equal(t, l1HeaderHash.Hex(), hash.Hex())
			return block, nil
		},
	}

	// Create the L1OriginProver using the constructor
	prover := NewL1OriginProver(mockL1Client, mockL2Client)

	// Call the method being tested
	encodedHeader, resultHeader, err := prover.ProveL1Origin(context.Background(), l1OracleAddress)
	require.NoError(t, err)

	// Verify the results
	assert.Equal(t, expectedEncodedHeader, encodedHeader)
	assert.Equal(t, block.Hash().Hex(), resultHeader.Hash().Hex())
	assert.Equal(t, block.Number().Uint64(), resultHeader.Number.Uint64())
}
