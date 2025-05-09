package provers

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/polymerdao/fallback_prover/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageProver_GetStorageAt(t *testing.T) {
	// Create test data
	address := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	slot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	expectedValue := "0x0000000000000000000000000000000000000000000000000000000000000123"

	// Create mock RPC client
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			// Check that we're calling the right method with the right arguments
			assert.Equal(t, "eth_getStorageAt", method)
			assert.Len(t, args, 3)
			testutil.RequireAddressEq(t, address, common.HexToAddress(args[0].(string)))
			assert.Equal(t, slot.Hex(), args[1])
			assert.Equal(t, "latest", args[2])

			// Set the result
			*(result.(*string)) = expectedValue
			return nil
		},
	}

	// Create the StorageProver
	prover := NewStorageProver(&testutil.MockEthClient{}, mockRPCClient)

	// Call the method being tested
	result, err := prover.GetStorageAt(context.Background(), address, slot, nil)
	require.NoError(t, err)

	// Verify the results
	assert.Equal(t, expectedValue, result)
}

func TestStorageProver_GetStorageProof(t *testing.T) {
	// Create test data
	address := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	slot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	value := big.NewInt(0x123)
	mockProof := testutil.MockStorageProofResult(t, address, slot, value)

	// Create mock RPC client
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			// Check that we're calling the right method with the right arguments
			assert.Equal(t, "eth_getProof", method)
			assert.Len(t, args, 3)
			testutil.RequireAddressEq(t, address, args[0].(common.Address))
			assert.Equal(t, []string{slot.Hex()}, args[1])
			assert.Equal(t, "latest", args[2])

			// Marshal the mock proof to JSON
			mockProofJSON, err := json.Marshal(mockProof)
			require.NoError(t, err)

			// Unmarshal it into the result
			err = json.Unmarshal(mockProofJSON, result)
			require.NoError(t, err)

			return nil
		},
	}

	// Create the StorageProver
	prover := NewStorageProver(&testutil.MockEthClient{}, mockRPCClient)

	// Call the method being tested
	result, err := prover.GetStorageProof(context.Background(), address, slot, nil)
	require.NoError(t, err)

	// Verify the results
	assert.Equal(t, address.Hex(), result.Address.Hex())
	assert.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", result.StorageHash.Hex())
	assert.Len(t, result.StorageProof, 1)
	assert.Equal(t, slot.Hex(), result.StorageProof[0].Key.Hex())
	assert.Equal(t, value, result.StorageProof[0].Value.ToInt())
}

func TestStorageProver_GenerateStorageProof(t *testing.T) {
	// Create test data
	address := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	slot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	value := big.NewInt(0x123)

	// Initialize storage proof result
	mockProof := testutil.MockStorageProofResult(t, address, slot, value)

	// Create mock RPC client
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			// Check that we're calling the right method with the right arguments
			assert.Equal(t, "eth_getProof", method)

			// Marshal the mock proof to JSON
			mockProofJSON, err := json.Marshal(mockProof)
			require.NoError(t, err)

			// Unmarshal it into the result
			err = json.Unmarshal(mockProofJSON, result)
			require.NoError(t, err)

			return nil
		},
	}

	// Create the StorageProver
	prover := NewStorageProver(&testutil.MockEthClient{}, mockRPCClient)

	// Call the method being tested
	storageProof, rlpEncodedContractAccount, accountProof, err := prover.GenerateStorageProof(
		context.Background(),
		address,
		slot,
		big.NewInt(3),
	)
	require.NoError(t, err)

	// Verify the results
	assert.Len(t, storageProof, 2)
	assert.NotNil(t, rlpEncodedContractAccount)
	assert.Len(t, accountProof, 2)

	// Verify the RLP encoded account is correct by decoding it
	var account Account
	err = rlp.DecodeBytes(rlpEncodedContractAccount, &account)
	require.NoError(t, err)

	assert.Equal(t, uint64(0), account.Nonce)
	assert.Equal(t, big.NewInt(0), account.Balance)
	assert.Equal(
		t,
		common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		account.Root,
	)
}
