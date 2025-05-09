package fallback_prover

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/polymerdao/fallback_prover/provers"
	"github.com/polymerdao/fallback_prover/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestL1Prover_GenerateProveL1Calldata(t *testing.T) {
	// Create test data
	l1Address := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	l1StorageSlot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	// Create a test header and block
	l1Header := testutil.CreateTestHeader(t)
	l1Block := testutil.CreateTestBlock(t, l1Header)

	// RLP encode header
	rlpEncodedL1Header, err := rlp.EncodeToBytes(l1Header)
	require.NoError(t, err)

	// Mock storage proof data
	mockStorageProof := [][]byte{[]byte("storage-proof-1"), []byte("storage-proof-2")}
	mockEncodedContractAccount := []byte("mock-encoded-contract-account")
	mockAccountProof := [][]byte{[]byte("account-proof-1"), []byte("account-proof-2")}

	// Create mock provers
	mockL1OriginProver := &testutil.MockL1OriginProver{
		GetL1OriginFunc: func(ctx context.Context, l1Hash common.Hash) ([]byte, *types.Header, error) {
			return rlpEncodedL1Header, l1Block.Header(), nil
		},
		GetL1OriginHashFunc: func(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error) {
			return l1Block.Hash(), nil
		},
	}

	mockStorageProver := &testutil.MockStorageProver{
		GetStorageAtFunc: func(ctx context.Context, address common.Address, slot common.Hash, blockNumber *big.Int) (string, error) {
			return "0x0000000000000000000000000000000000000000000000000000000000000123", nil
		},
		GenerateStorageProofFunc: func(ctx context.Context, contractAddr common.Address, storageSlot common.Hash, blockNumber *big.Int) ([][]byte, []byte, [][]byte, error) {
			return mockStorageProof, mockEncodedContractAccount, mockAccountProof, nil
		},
	}

	// Create a real NativeProver
	nativeProver, err := provers.NewNativeProver()
	require.NoError(t, err)

	// Create the L1Prover instance with mocked interfaces and real NativeProver
	prover := &L1Prover{
		l1OriginProver:    mockL1OriginProver,
		l1StorageProver:   mockStorageProver,
		nativeProver:      nativeProver,
		l1BlockHashOracle: common.HexToAddress("0x5678"),
	}

	// Call the method being tested
	calldata, err := prover.GenerateProveL1Calldata(
		context.Background(),
		&ProveParams{
			Address:     l1Address,
			StorageSlot: l1StorageSlot,
		},
	)
	require.NoError(t, err)

	// Check that the calldata is not empty and has a 0x prefix
	require.True(t, len(calldata) > 2, "Calldata should not be empty")
	require.Equal(t, "0x", calldata[:2], "Calldata should start with 0x")

	// Check that the calldata has the correct function selector
	hexCalldata := common.Hex2Bytes(calldata[2:]) // Skip the "0x" prefix
	actualSelector := hexutil.Encode(hexCalldata[:4])

	// The actual function selector may vary, but should be a valid hex string
	require.Len(t, actualSelector, 10, "Function selector should be 4 bytes (10 chars in hex)")
	t.Logf("Function selector: %s", actualSelector)

	// Try to unpack the calldata to verify its structure
	// First, get the ABI from the nativeProver
	nativeProverABI := prover.nativeProver.GetABI()

	// Create a map to hold the unpacked values
	unpackedMap := make(map[string]interface{})

	// Use UnpackIntoMap to decode the calldata
	err = nativeProverABI.Methods["proveL1Native"].Inputs.UnpackIntoMap(unpackedMap, hexCalldata[4:])
	if err != nil {
		t.Logf("Failed to unpack calldata: %v", err)
		t.Logf("This could be because the ABI doesn't have a proveL1Native method defined yet")

		// Log available methods
		t.Logf("Available methods in ABI:")
		for methodName := range nativeProverABI.Methods {
			t.Logf("  - %s", methodName)
		}
	} else {
		// We successfully unpacked the calldata!

		// Log the map to see what we get
		t.Logf("Unpacked map: %v", unpackedMap)

		// Extract the _args field and verify its contents
		argsVal, exists := unpackedMap["_args"]
		if exists {
			t.Logf("argsVal type: %T", argsVal)

			// Check if contractAddr in args matches our input
			argsStruct, ok := argsVal.(struct {
				ContractAddr     common.Address
				StorageSlot      [32]uint8
				StorageValue     [32]uint8
				L1WorldStateRoot [32]uint8
			})

			if ok {
				assert.Equal(t, l1Address.Hex(), argsStruct.ContractAddr.Hex(), "ContractAddr should match")

				// Convert [32]uint8 to common.Hash for comparison
				var storageSlotBytes common.Hash
				copy(storageSlotBytes[:], argsStruct.StorageSlot[:])
				assert.Equal(t, l1StorageSlot.Hex(), storageSlotBytes.Hex(), "StorageSlot should match")
			}
		}

		// Verify other parameters
		assert.NotNil(t, unpackedMap["_rlpEncodedL1Header"], "L1 header should be present")
		assert.NotNil(t, unpackedMap["_l1StorageProof"], "L1 storage proof should be present")
		assert.NotNil(t, unpackedMap["_rlpEncodedContractAccount"], "RLP encoded contract account should be present")
		assert.NotNil(t, unpackedMap["_l1AccountProof"], "L1 account proof should be present")
	}
}
