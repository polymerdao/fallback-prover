package provers

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/polymerdao/fallback_prover/types"
)

// convertToProveScalarArgs converts the unpacked ABI data to ProveScalarArgs struct
// without using reflection
func convertToProveScalarArgs(t *testing.T, argsVal interface{}) types.ProveScalarArgs {
	// Cast argsVal to the concrete type that the ABI outputs when unpacking
	// This will be a struct with the same field names as ProveScalarArgs but with different type for hash fields
	structType, ok := argsVal.(struct {
		ChainID          *big.Int       `json:"chainID"`
		ContractAddr     common.Address `json:"contractAddr"`
		StorageSlot      [32]uint8      `json:"storageSlot"`
		StorageValue     [32]uint8      `json:"storageValue"`
		L2WorldStateRoot [32]uint8      `json:"l2WorldStateRoot"`
	})

	if !ok {
		t.Logf("Unexpected structure for argsVal: %T", argsVal)
		t.Fatalf("Failed to cast unpacked args to expected struct type: %T", argsVal)
	}

	// Convert [32]uint8 to common.Hash
	var storageSlot, storageValue, l2WorldStateRoot common.Hash
	copy(storageSlot[:], structType.StorageSlot[:])
	copy(storageValue[:], structType.StorageValue[:])
	copy(l2WorldStateRoot[:], structType.L2WorldStateRoot[:])

	// Construct a new ProveScalarArgs with the extracted values
	return types.ProveScalarArgs{
		ChainID:          structType.ChainID,
		ContractAddr:     structType.ContractAddr,
		StorageSlot:      storageSlot,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2WorldStateRoot,
	}
}

// convertToUpdateL2ConfigArgs converts the unpacked ABI data to UpdateL2ConfigArgs struct
// without using reflection
func convertToUpdateL2ConfigArgs(t *testing.T, argsVal interface{}) types.UpdateL2ConfigArgs {
	// Cast argsVal to the concrete type that the ABI outputs when unpacking
	// This will be a struct with the same field names as UpdateL2ConfigArgs and the nested L2Configuration
	structType, ok := argsVal.(struct {
		Config struct {
			Prover               common.Address   `json:"prover"`
			Addresses            []common.Address `json:"addresses"`
			StorageSlots         []*big.Int       `json:"storageSlots"`
			VersionNumber        *big.Int         `json:"versionNumber"`
			FinalityDelaySeconds *big.Int         `json:"finalityDelaySeconds"`
			L2Type               uint8            `json:"l2Type"`
		} `json:"config"`
		L1StorageProof                [][]byte `json:"l1StorageProof"`
		RlpEncodedRegistryAccountData []byte   `json:"rlpEncodedRegistryAccountData"`
		L1RegistryProof               [][]byte `json:"l1RegistryProof"`
	})

	if !ok {
		t.Logf("Unexpected structure for argsVal: %T", argsVal)
		t.Fatalf("Failed to cast unpacked args to expected struct type: %T", argsVal)
	}

	// Construct a new L2Configuration with the extracted values
	config := types.L2Configuration{
		Prover:               structType.Config.Prover,
		Addresses:            structType.Config.Addresses,
		StorageSlots:         structType.Config.StorageSlots,
		VersionNumber:        structType.Config.VersionNumber,
		FinalityDelaySeconds: structType.Config.FinalityDelaySeconds,
		L2Type:               types.L2Type(structType.Config.L2Type),
	}

	// Construct a new UpdateL2ConfigArgs with the extracted values
	return types.UpdateL2ConfigArgs{
		Config:                        config,
		L1StorageProof:                structType.L1StorageProof,
		RlpEncodedRegistryAccountData: structType.RlpEncodedRegistryAccountData,
		L1RegistryProof:               structType.L1RegistryProof,
	}
}

func TestNativeProver_EncodeProveNativeCalldata(t *testing.T) {
	// Test data for UpdateL2ConfigArgs
	config := types.L2Configuration{
		Prover:               common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432"),
		Addresses:            []common.Address{common.HexToAddress("0xdeadbeef")},
		StorageSlots:         []*big.Int{big.NewInt(123)},
		VersionNumber:        big.NewInt(1),
		FinalityDelaySeconds: big.NewInt(300),
		L2Type:               types.OPStackCannon,
	}
	l1StorageProof := [][]byte{
		[]byte("l1-storage-proof-1"),
		[]byte("l1-storage-proof-2"),
	}
	rlpEncodedRegistryAccountData := []byte("mock-rlp-encoded-registry-account-data")
	l1RegistryProof := [][]byte{
		[]byte("l1-registry-proof-1"),
		[]byte("l1-registry-proof-2"),
	}
	updateArgs := types.UpdateL2ConfigArgs{
		Config:                        config,
		L1StorageProof:                l1StorageProof,
		RlpEncodedRegistryAccountData: rlpEncodedRegistryAccountData,
		L1RegistryProof:               l1RegistryProof,
	}

	// Test data for ProveScalarArgs
	chainID := big.NewInt(42161)
	contractAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	storageSlot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	storageValue := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000123")
	l2WorldStateRoot := common.HexToHash("0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba")
	proveArgs := types.ProveScalarArgs{
		ChainID:          chainID,
		ContractAddr:     contractAddr,
		StorageSlot:      storageSlot,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2WorldStateRoot,
	}

	// Common test data
	rlpEncodedL1Header := []byte("mock-rlp-encoded-l1-header")
	rlpEncodedL2Header := []byte("mock-rlp-encoded-l2-header")
	settledStateProof := []byte("mock-settled-state-proof")
	l2StorageProof := [][]byte{
		[]byte("storage-proof-1"),
		[]byte("storage-proof-2"),
	}
	rlpEncodedContractAccount := []byte("mock-rlp-encoded-contract-account")
	l2AccountProof := [][]byte{
		[]byte("account-proof-1"),
		[]byte("account-proof-2"),
	}

	// Create the NativeProver
	prover, err := NewNativeProver()
	require.NoError(t, err)

	abi := prover.GetABI()

	// Call the method being tested
	calldata, err := prover.EncodeProveNativeCalldata(
		updateArgs,
		proveArgs,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
		settledStateProof,
		l2StorageProof,
		rlpEncodedContractAccount,
		l2AccountProof,
	)
	require.NoError(t, err)

	// Check the function selector
	actualSelector := hexutil.Encode(calldata[:4])
	t.Logf("Function selector for proveNative: %s", actualSelector)
	expectedFunctionSelector := hexutil.Encode(abi.Methods["proveNative"].ID)
	assert.Equal(t, expectedFunctionSelector, actualSelector, "Calldata should start with the function selector for proveNative")

	// Create a map to hold the unpacked values
	unpackedMap := make(map[string]interface{})

	// Use UnpackIntoMap to get the struct values
	err = prover.abi.Methods["proveNative"].Inputs.UnpackIntoMap(unpackedMap, calldata[4:])
	require.NoError(t, err, "Failed to unpack calldata")

	// Log the map
	t.Logf("Unpacked map: %v", unpackedMap)

	// Extract and verify the update args
	updateArgsVal, exists := unpackedMap["_updateArgs"]
	require.True(t, exists, "Expected _updateArgs to exist in unpacked map")

	// Use our helper to extract the update args
	updateArgsStruct := convertToUpdateL2ConfigArgs(t, updateArgsVal)
	// Verify updateArgs values
	assert.Equal(t, config.Prover.Hex(), updateArgsStruct.Config.Prover.Hex(), "Prover address should match")
	assert.Equal(t, len(config.Addresses), len(updateArgsStruct.Config.Addresses), "Addresses length should match")
	assert.Equal(t, config.Addresses[0].Hex(), updateArgsStruct.Config.Addresses[0].Hex(), "First address should match")
	assert.Equal(t, config.StorageSlots[0].String(), updateArgsStruct.Config.StorageSlots[0].String(), "First storage slot should match")
	assert.Equal(t, config.VersionNumber.String(), updateArgsStruct.Config.VersionNumber.String(), "Version number should match")
	assert.Equal(t, config.FinalityDelaySeconds.String(), updateArgsStruct.Config.FinalityDelaySeconds.String(), "Finality delay should match")
	assert.Equal(t, config.L2Type, updateArgsStruct.Config.L2Type, "L2Type should match")

	// Extract and verify the prove args
	proveArgsVal, exists := unpackedMap["_proveArgs"]
	require.True(t, exists, "Expected _proveArgs to exist in unpacked map")

	// Use our helper to extract the prove args
	extractedProveArgs := convertToProveScalarArgs(t, proveArgsVal)

	// Verify proveArgs values
	assert.Equal(t, chainID.String(), extractedProveArgs.ChainID.String(), "ChainID should match")
	assert.Equal(t, contractAddr.Hex(), extractedProveArgs.ContractAddr.Hex(), "ContractAddr should match")
	assert.Equal(t, storageSlot.Hex(), extractedProveArgs.StorageSlot.Hex(), "StorageSlot should match")
	assert.Equal(t, storageValue.Hex(), extractedProveArgs.StorageValue.Hex(), "StorageValue should match")
	assert.Equal(t, l2WorldStateRoot.Hex(), extractedProveArgs.L2WorldStateRoot.Hex(), "L2WorldStateRoot should match")

	// Check the rest of the arguments
	assert.Equal(t, rlpEncodedL1Header, unpackedMap["_rlpEncodedL1Header"].([]byte), "RlpEncodedL1Header should match")
	assert.Equal(t, rlpEncodedL2Header, unpackedMap["_rlpEncodedL2Header"].([]byte), "RlpEncodedL2Header should match")
	assert.Equal(t, settledStateProof, unpackedMap["_settledStateProof"].([]byte), "SettledStateProof should match")
}
