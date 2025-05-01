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
	types2 "github.com/polymerdao/fallback_prover/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProver_GenerateProveCalldata(t *testing.T) {
	// Create test data
	srcL2ChainID := uint64(10) // Optimism chain ID
	srcAddress := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	srcStorageSlot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	// Create a test header and block
	l1Header := testutil.CreateTestHeader(t)
	l1Block := testutil.CreateTestBlock(t, l1Header)
	l2Header := testutil.CreateTestHeader(t)

	// RLP encode headers
	rlpEncodedL1Header, err := rlp.EncodeToBytes(l1Header)
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
	mockL1OriginProver := &testutil.MockL1OriginProver{
		GetL1OriginHashFunc: func(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error) {
			return l1Block.Hash(), nil
		},
		GetL1OriginFunc: func(ctx context.Context, l1OriginHash common.Hash) ([]byte, *types.Header, error) {
			return rlpEncodedL1Header, l1Block.Header(), nil
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
		GenerateSettledStateProofFunc: func(ctx context.Context, l1BlockNumber *big.Int, config *types2.L2ConfigInfo) ([]byte, *types.Header, error) {
			return mockSettledStateProof, l2Header, nil
		},
	}

	nativeProver, err := provers.NewNativeProver()
	require.NoError(t, err)

	// Create the Prover instance with mocked interfaces
	prover := &Prover{
		l1OriginProver:     mockL1OriginProver,
		nativeProver:       nativeProver,
		l2StorageProver:    mockStorageProver,
		settledStateProver: mockBedrockProver,
		l2Config:           testConfig,
		l1BlockHashOracle:  common.HexToAddress("0x5678"),
		srcChainID:         big.NewInt(int64(srcL2ChainID)), // Initialize the srcChainID field
	}

	// Call the method being tested
	calldata, err := prover.GenerateProveCalldata(
		context.Background(),
		&ProveParams{
			Address:     srcAddress,
			StorageSlot: srcStorageSlot,
		},
	)
	require.NoError(t, err)

	// Check that the calldata has the correct function selector
	hexCalldata := common.Hex2Bytes(calldata[2:]) // Skip the "0x" prefix
	actualSelector := hexutil.Encode(hexCalldata[:4])
	expectedSelector := "0x8d1f227a" // This matches the actual function selector in our updated ABI
	assert.Equal(t, expectedSelector, actualSelector, "Calldata should start with the correct function selector")

	// Try to unpack the calldata to compare the arguments individually
	// First, get the ABI from the nativeProver
	nativeProverABI := prover.nativeProver.(*provers.NativeProver).GetABI()

	// Create a map to hold the unpacked values
	unpackedMap := make(map[string]interface{})

	// Use UnpackIntoMap which is more flexible and doesn't require matching the Solidity parameter names
	err = nativeProverABI.Methods["prove"].Inputs.UnpackIntoMap(unpackedMap, hexCalldata[4:])
	require.NoError(t, err, "Failed to unpack calldata")

	// Log the map to see what we get
	t.Logf("Unpacked map: %v", unpackedMap)

	// Extract the arguments from the map and verify them
	argsVal, exists := unpackedMap["_args"]
	require.True(t, exists, "Expected _args to exist in unpacked map")

	// Log the struct type to help debug
	t.Logf("argsVal type: %T", argsVal)

	// Extract the fields with a type assertion matching the actual struct type
	structType, ok := argsVal.(struct {
		ChainID          *big.Int       `json:"chainID"`
		ContractAddr     common.Address `json:"contractAddr"`
		StorageSlot      [32]uint8      `json:"storageSlot"`
		StorageValue     [32]uint8      `json:"storageValue"`
		L2WorldStateRoot [32]uint8      `json:"l2WorldStateRoot"`
	})
	require.True(t, ok, "Failed to cast to expected struct type")

	// Verify the values directly against our expectations
	assert.Equal(t, big.NewInt(int64(srcL2ChainID)).String(), structType.ChainID.String(), "ChainID should match")
	assert.Equal(t, srcAddress.Hex(), structType.ContractAddr.Hex(), "ContractAddr should match")

	// Convert [32]uint8 to common.Hash for comparison
	var storageSlotBytes common.Hash
	copy(storageSlotBytes[:], structType.StorageSlot[:])
	assert.Equal(t, srcStorageSlot.Hex(), storageSlotBytes.Hex(), "StorageSlot should match")

	// Check for the presence of other key data
	assert.NotNil(t, unpackedMap["_rlpEncodedL1Header"], "L1 header should be present")
	assert.NotNil(t, unpackedMap["_rlpEncodedL2Header"], "L2 header should be present")
	assert.NotNil(t, unpackedMap["_settledStateProof"], "Settled state proof should be present")

	storageProofFromMap, ok := unpackedMap["_l2StorageProof"].([][]byte)
	require.True(t, ok, "Expected _l2StorageProof to be a [][]byte")
	assert.NotEmpty(t, storageProofFromMap, "L2 storage proof should be present")

	assert.NotNil(t, unpackedMap["_rlpEncodedContractAccount"], "RLP encoded contract account should be present")

	accountProofFromMap, ok := unpackedMap["_l2AccountProof"].([][]byte)
	require.True(t, ok, "Expected _l2AccountProof to be a [][]byte")
	assert.NotEmpty(t, accountProofFromMap, "L2 account proof should be present")
}

func TestProver_GenerateUpdateAndProveCalldata(t *testing.T) {
	// Create test data
	srcL2ChainID := uint64(10) // Optimism chain ID
	srcAddress := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	srcStorageSlot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	// Create a test header and block
	l1Header := testutil.CreateTestHeader(t)
	l1Block := testutil.CreateTestBlock(t, l1Header)
	l2Header := testutil.CreateTestHeader(t)

	// RLP encode headers
	rlpEncodedL1Header, err := rlp.EncodeToBytes(l1Header)
	require.NoError(t, err)

	// Mock settled state proof data
	mockSettledStateProof := []byte("mock-settled-state-proof")
	mockStorageProof := [][]byte{[]byte("storage-proof-1"), []byte("storage-proof-2")}
	mockEncodedContractAccount := []byte("mock-encoded-contract-account")
	mockAccountProof := [][]byte{[]byte("account-proof-1"), []byte("account-proof-2")}

	// Create mock L1 storage proof data for UpdateL2ConfigArgs
	mockL1StorageProof := [][]byte{[]byte("l1-storage-proof-1"), []byte("l1-storage-proof-2")}
	mockEncodedRegistryAccount := []byte("mock-encoded-registry-account")
	mockL1RegistryProof := [][]byte{[]byte("l1-registry-proof-1"), []byte("l1-registry-proof-2")}

	// Create a test config using our testutil.L2ConfigInfo
	testConfig := &types2.L2ConfigInfo{
		ConfigType: "OPStackBedrock",
		Addresses: []common.Address{
			common.HexToAddress("0x1234"),
		},
		StorageSlots: []uint64{0x123},
	}

	// Create UpdateL2ConfigArgs
	l2Config := types2.L2Configuration{
		Prover:               common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432"),
		Addresses:            []common.Address{common.HexToAddress("0xdeadbeef")},
		StorageSlots:         []*big.Int{big.NewInt(123)},
		VersionNumber:        big.NewInt(1),
		FinalityDelaySeconds: big.NewInt(300),
		L2Type:               types2.OPStackBedrock,
	}

	// Create mock provers
	mockL1OriginProver := &testutil.MockL1OriginProver{
		GetL1OriginHashFunc: func(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error) {
			return l1Block.Hash(), nil
		},
		GetL1OriginFunc: func(ctx context.Context, l1OriginHash common.Hash) ([]byte, *types.Header, error) {
			return rlpEncodedL1Header, l1Block.Header(), nil
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
		GenerateSettledStateProofFunc: func(ctx context.Context, l1BlockNumber *big.Int, config *types2.L2ConfigInfo) ([]byte, *types.Header, error) {
			return mockSettledStateProof, l2Header, nil
		},
	}

	nativeProver, err := provers.NewNativeProver()
	require.NoError(t, err)

	// Create the Prover instance with mocked interfaces
	prover := &Prover{
		l1OriginProver:     mockL1OriginProver,
		nativeProver:       nativeProver,
		l2StorageProver:    mockStorageProver,
		settledStateProver: mockBedrockProver,
		l2Config:           testConfig,
		l1BlockHashOracle:  common.HexToAddress("0x5678"),
		srcChainID:         big.NewInt(int64(srcL2ChainID)), // Initialize the srcChainID field
		configProof: &types2.UpdateL2ConfigArgs{
			Config:                        l2Config,
			L1StorageProof:                mockL1StorageProof,
			RlpEncodedRegistryAccountData: mockEncodedRegistryAccount,
			L1RegistryProof:               mockL1RegistryProof,
		},
	}

	// Call the method being tested
	calldata, err := prover.GenerateUpdateAndProveCalldata(
		context.Background(),
		&ProveParams{
			Address:     srcAddress,
			StorageSlot: srcStorageSlot,
		},
	)
	require.NoError(t, err)

	// Check that the calldata has the correct function selector
	hexCalldata := common.Hex2Bytes(calldata[2:]) // Skip the "0x" prefix
	actualSelector := hexutil.Encode(hexCalldata[:4])
	expectedSelector := "0xc1ed98af" // This should match the actual selector for updateAndProve
	assert.Equal(t, expectedSelector, actualSelector, "Calldata should start with the correct function selector")

	// Try to unpack the calldata to compare the arguments individually
	// First, get the ABI from the nativeProver
	nativeProverABI := prover.nativeProver.(*provers.NativeProver).GetABI()

	// Create a map to hold the unpacked values
	unpackedMap := make(map[string]interface{})

	// Use UnpackIntoMap which is more flexible and doesn't require matching the Solidity parameter names
	err = nativeProverABI.Methods["updateAndProve"].Inputs.UnpackIntoMap(unpackedMap, hexCalldata[4:])
	require.NoError(t, err, "Failed to unpack calldata")

	// Log the map to see what we get
	t.Logf("Unpacked map: %v", unpackedMap)

	// Verify the _updateArgs
	updateArgsVal, exists := unpackedMap["_updateArgs"]
	require.True(t, exists, "Expected _updateArgs to exist in unpacked map")

	// Log the updateArgsVal type to help debug
	t.Logf("updateArgsVal type: %T", updateArgsVal)

	// Extract the fields from updateArgsVal with a type assertion matching the UpdateL2ConfigArgs struct
	updateArgsStruct, ok := updateArgsVal.(struct {
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
	require.True(t, ok, "Failed to cast updateArgsVal to expected struct type")

	// Verify the updateArgs fields match expected values
	assert.Equal(t, common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432").Hex(), updateArgsStruct.Config.Prover.Hex(), "Prover address should match")
	assert.Equal(t, common.HexToAddress("0xdeadbeef").Hex(), updateArgsStruct.Config.Addresses[0].Hex(), "Config address should match")
	assert.Equal(t, big.NewInt(123).String(), updateArgsStruct.Config.StorageSlots[0].String(), "StorageSlot should match")
	assert.Equal(t, big.NewInt(1).String(), updateArgsStruct.Config.VersionNumber.String(), "VersionNumber should match")
	assert.Equal(t, big.NewInt(300).String(), updateArgsStruct.Config.FinalityDelaySeconds.String(), "FinalityDelaySeconds should match")
	assert.Equal(t, uint8(types2.OPStackBedrock), updateArgsStruct.Config.L2Type, "L2Type should match")
	assert.NotEmpty(t, updateArgsStruct.L1StorageProof, "L1StorageProof should be present")
	assert.NotEmpty(t, updateArgsStruct.RlpEncodedRegistryAccountData, "RlpEncodedRegistryAccountData should be present")
	assert.NotEmpty(t, updateArgsStruct.L1RegistryProof, "L1RegistryProof should be present")

	// Verify the _proveArgs
	proveArgsVal, exists := unpackedMap["_proveArgs"]
	require.True(t, exists, "Expected _proveArgs to exist in unpacked map")

	// Extract the proveArgs fields with type assertion
	proveArgsStruct, ok := proveArgsVal.(struct {
		ChainID          *big.Int       `json:"chainID"`
		ContractAddr     common.Address `json:"contractAddr"`
		StorageSlot      [32]uint8      `json:"storageSlot"`
		StorageValue     [32]uint8      `json:"storageValue"`
		L2WorldStateRoot [32]uint8      `json:"l2WorldStateRoot"`
	})
	require.True(t, ok, "Failed to cast proveArgsVal to expected struct type")

	// Verify the core values
	assert.Equal(t, big.NewInt(int64(srcL2ChainID)).String(), proveArgsStruct.ChainID.String(), "ChainID should match")
	assert.Equal(t, srcAddress.Hex(), proveArgsStruct.ContractAddr.Hex(), "ContractAddr should match")

	// Convert [32]uint8 to common.Hash for comparison
	var storageSlotBytes common.Hash
	copy(storageSlotBytes[:], proveArgsStruct.StorageSlot[:])
	assert.Equal(t, srcStorageSlot.Hex(), storageSlotBytes.Hex(), "StorageSlot should match")

	// Check for the presence of other key data
	assert.NotNil(t, unpackedMap["_rlpEncodedL1Header"], "L1 header should be present")
	assert.NotNil(t, unpackedMap["_rlpEncodedL2Header"], "L2 header should be present")
	assert.NotNil(t, unpackedMap["_settledStateProof"], "Settled state proof should be present")

	storageProofFromMap, ok := unpackedMap["_l2StorageProof"].([][]byte)
	require.True(t, ok, "Expected _l2StorageProof to be a [][]byte")
	assert.NotEmpty(t, storageProofFromMap, "L2 storage proof should be present")

	assert.NotNil(t, unpackedMap["_rlpEncodedContractAccount"], "RLP encoded contract account should be present")

	accountProofFromMap, ok := unpackedMap["_l2AccountProof"].([][]byte)
	require.True(t, ok, "Expected _l2AccountProof to be a [][]byte")
	assert.NotEmpty(t, accountProofFromMap, "L2 account proof should be present")
}

func TestProver_GenerateConfigureAndProveCalldata(t *testing.T) {
	// Create test data
	srcL2ChainID := uint64(10) // Optimism chain ID
	srcAddress := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	srcStorageSlot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	// Create a test header and block
	l1Header := testutil.CreateTestHeader(t)
	l1Block := testutil.CreateTestBlock(t, l1Header)
	l2Header := testutil.CreateTestHeader(t)

	// RLP encode headers
	rlpEncodedL1Header, err := rlp.EncodeToBytes(l1Header)
	require.NoError(t, err)

	// Mock settled state proof data
	mockSettledStateProof := []byte("mock-settled-state-proof")
	mockStorageProof := [][]byte{[]byte("storage-proof-1"), []byte("storage-proof-2")}
	mockEncodedContractAccount := []byte("mock-encoded-contract-account")
	mockAccountProof := [][]byte{[]byte("account-proof-1"), []byte("account-proof-2")}

	// Create mock L1 storage proof data for UpdateL2ConfigArgs
	mockL1StorageProof := [][]byte{[]byte("l1-storage-proof-1"), []byte("l1-storage-proof-2")}
	mockEncodedRegistryAccount := []byte("mock-encoded-registry-account")
	mockL1RegistryProof := [][]byte{[]byte("l1-registry-proof-1"), []byte("l1-registry-proof-2")}

	// Create a test config using our testutil.L2ConfigInfo
	testConfig := &types2.L2ConfigInfo{
		ConfigType: "OPStackCannon",
		Addresses: []common.Address{
			common.HexToAddress("0x1234"),
		},
		StorageSlots: []uint64{0x123},
	}

	// Create UpdateL2ConfigArgs with Cannon L2Type
	l2Config := types2.L2Configuration{
		Prover:               common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432"),
		Addresses:            []common.Address{common.HexToAddress("0xdeadbeef")},
		StorageSlots:         []*big.Int{big.NewInt(123)},
		VersionNumber:        big.NewInt(1),
		FinalityDelaySeconds: big.NewInt(300),
		L2Type:               types2.OPStackCannon,
	}

	// Create mock provers
	mockL1OriginProver := &testutil.MockL1OriginProver{
		GetL1OriginHashFunc: func(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error) {
			return l1Block.Hash(), nil
		},
		GetL1OriginFunc: func(ctx context.Context, l1OriginHash common.Hash) ([]byte, *types.Header, error) {
			return rlpEncodedL1Header, l1Block.Header(), nil
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

	mockCannonProver := &testutil.MockOPStackCannonProver{
		GenerateSettledStateProofFunc: func(ctx context.Context, l1BlockNumber *big.Int, config *types2.L2ConfigInfo) ([]byte, *types.Header, error) {
			return mockSettledStateProof, l2Header, nil
		},
	}

	nativeProver, err := provers.NewNativeProver()
	require.NoError(t, err)

	// Create the Prover instance with mocked interfaces
	prover := &Prover{
		l1OriginProver:     mockL1OriginProver,
		nativeProver:       nativeProver,
		l2StorageProver:    mockStorageProver,
		settledStateProver: mockCannonProver,
		l2Config:           testConfig,
		l1BlockHashOracle:  common.HexToAddress("0x5678"),
		srcChainID:         big.NewInt(int64(srcL2ChainID)), // Initialize the srcChainID field
		configProof: &types2.UpdateL2ConfigArgs{
			Config:                        l2Config,
			L1StorageProof:                mockL1StorageProof,
			RlpEncodedRegistryAccountData: mockEncodedRegistryAccount,
			L1RegistryProof:               mockL1RegistryProof,
		},
	}

	// Call the method being tested
	calldata, err := prover.GenerateConfigureAndProveCalldata(
		context.Background(),
		&ProveParams{
			Address:     srcAddress,
			StorageSlot: srcStorageSlot,
		},
	)
	require.NoError(t, err)

	// Check that the calldata has the correct function selector
	hexCalldata := common.Hex2Bytes(calldata[2:]) // Skip the "0x" prefix
	actualSelector := hexutil.Encode(hexCalldata[:4])
	expectedSelector := "0x3c873bb2" // This should match the actual selector for configureAndProve
	assert.Equal(t, expectedSelector, actualSelector, "Calldata should start with the correct function selector")

	// Try to unpack the calldata to compare the arguments individually
	// First, get the ABI from the nativeProver
	nativeProverABI := prover.nativeProver.(*provers.NativeProver).GetABI()

	// Create a map to hold the unpacked values
	unpackedMap := make(map[string]interface{})

	// Use UnpackIntoMap which is more flexible and doesn't require matching the Solidity parameter names
	err = nativeProverABI.Methods["configureAndProve"].Inputs.UnpackIntoMap(unpackedMap, hexCalldata[4:])
	require.NoError(t, err, "Failed to unpack calldata")

	// Log the map to see what we get
	t.Logf("Unpacked map: %v", unpackedMap)

	// Verify the _updateArgs
	updateArgsVal, exists := unpackedMap["_updateArgs"]
	require.True(t, exists, "Expected _updateArgs to exist in unpacked map")

	// Log the updateArgsVal type to help debug
	t.Logf("updateArgsVal type: %T", updateArgsVal)

	// Extract the fields from updateArgsVal with a type assertion matching the UpdateL2ConfigArgs struct
	updateArgsStruct, ok := updateArgsVal.(struct {
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
	require.True(t, ok, "Failed to cast updateArgsVal to expected struct type")

	// Verify the updateArgs fields match expected values
	assert.Equal(t, common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432").Hex(), updateArgsStruct.Config.Prover.Hex(), "Prover address should match")
	assert.Equal(t, common.HexToAddress("0xdeadbeef").Hex(), updateArgsStruct.Config.Addresses[0].Hex(), "Config address should match")
	assert.Equal(t, big.NewInt(123).String(), updateArgsStruct.Config.StorageSlots[0].String(), "StorageSlot should match")
	assert.Equal(t, big.NewInt(1).String(), updateArgsStruct.Config.VersionNumber.String(), "VersionNumber should match")
	assert.Equal(t, big.NewInt(300).String(), updateArgsStruct.Config.FinalityDelaySeconds.String(), "FinalityDelaySeconds should match")
	assert.Equal(t, uint8(types2.OPStackCannon), updateArgsStruct.Config.L2Type, "L2Type should match")
	assert.NotEmpty(t, updateArgsStruct.L1StorageProof, "L1StorageProof should be present")
	assert.NotEmpty(t, updateArgsStruct.RlpEncodedRegistryAccountData, "RlpEncodedRegistryAccountData should be present")
	assert.NotEmpty(t, updateArgsStruct.L1RegistryProof, "L1RegistryProof should be present")

	// Verify the _proveArgs
	proveArgsVal, exists := unpackedMap["_proveArgs"]
	require.True(t, exists, "Expected _proveArgs to exist in unpacked map")

	// Extract the proveArgs fields with type assertion
	proveArgsStruct, ok := proveArgsVal.(struct {
		ChainID          *big.Int       `json:"chainID"`
		ContractAddr     common.Address `json:"contractAddr"`
		StorageSlot      [32]uint8      `json:"storageSlot"`
		StorageValue     [32]uint8      `json:"storageValue"`
		L2WorldStateRoot [32]uint8      `json:"l2WorldStateRoot"`
	})
	require.True(t, ok, "Failed to cast proveArgsVal to expected struct type")

	// Verify the core values
	assert.Equal(t, big.NewInt(int64(srcL2ChainID)).String(), proveArgsStruct.ChainID.String(), "ChainID should match")
	assert.Equal(t, srcAddress.Hex(), proveArgsStruct.ContractAddr.Hex(), "ContractAddr should match")

	// Convert [32]uint8 to common.Hash for comparison
	var storageSlotBytes common.Hash
	copy(storageSlotBytes[:], proveArgsStruct.StorageSlot[:])
	assert.Equal(t, srcStorageSlot.Hex(), storageSlotBytes.Hex(), "StorageSlot should match")

	// Check for the presence of other key data
	assert.NotNil(t, unpackedMap["_rlpEncodedL1Header"], "L1 header should be present")
	assert.NotNil(t, unpackedMap["_rlpEncodedL2Header"], "L2 header should be present")
	assert.NotNil(t, unpackedMap["_settledStateProof"], "Settled state proof should be present")

	storageProofFromMap, ok := unpackedMap["_l2StorageProof"].([][]byte)
	require.True(t, ok, "Expected _l2StorageProof to be a [][]byte")
	assert.NotEmpty(t, storageProofFromMap, "L2 storage proof should be present")

	assert.NotNil(t, unpackedMap["_rlpEncodedContractAccount"], "RLP encoded contract account should be present")

	accountProofFromMap, ok := unpackedMap["_l2AccountProof"].([][]byte)
	require.True(t, ok, "Expected _l2AccountProof to be a [][]byte")
	assert.NotEmpty(t, accountProofFromMap, "L2 account proof should be present")
}
