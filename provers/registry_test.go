package provers

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/polymerdao/fallback_prover/testutil"
	"github.com/polymerdao/fallback_prover/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryProver_GetL2Configuration(t *testing.T) {
	// Create test data
	chainID := uint64(42161) // Arbitrum One Chain ID
	addresses := []common.Address{
		common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
	}
	slots := []*big.Int{
		big.NewInt(1),
		big.NewInt(2),
	}
	configType := "OPStackBedrock"

	// Get the Registry ABI for testing
	registryABI, err := getRegistryABI()
	require.NoError(t, err)

	// Print all method signatures for debugging
	for name, method := range registryABI.Methods {
		t.Logf("Method: %s, Sig: 0x%x", name, method.ID)
	}

	// Create mock eth client that implements our IEthClient
	mockEthClient := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the right contract
			testutil.RequireAddressEq(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"), *msg.To)

			// Determine which method is being called by looking at the first 4 bytes of the calldata
			methodSig := hexutil.Encode(msg.Data[:4])
			t.Logf("Called method with signature: %s", methodSig)

			switch methodSig {
			case "0x5338efd4": // getL2ConfigType(uint256) - Updated method signature
				// Convert string to uint8 for the enum
				var configTypeUint8 uint8
				if configType == "OPStackBedrock" {
					configTypeUint8 = 1
				} else if configType == "OPStackCannon" {
					configTypeUint8 = 2
				} else if configType == "Arbitrum" {
					configTypeUint8 = 3
				}

				// Pack the method output with the uint8
				outputs := []interface{}{configTypeUint8}
				packed, err := registryABI.Methods["getL2ConfigType"].Outputs.Pack(outputs...)
				require.NoError(t, err)
				return packed, nil
			case "0x974ec7fc": // getL2ConfigAddresses(uint256)
				// Correctly pack the address array according to the method output signature
				return registryABI.Methods["getL2ConfigAddresses"].Outputs.Pack(addresses)
			case "0xa4f4ae39": // getL2ConfigStorageSlots(uint256)
				// Correctly pack the big int array according to the method output signature
				return registryABI.Methods["getL2ConfigStorageSlots"].Outputs.Pack(slots)
			default:
				t.Fatalf("unexpected method signature: %s", methodSig)
				return nil, nil
			}
		},
	}

	// Create mock RPC client that implements our IRPCClient
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			return nil
		},
	}

	// Create the RegistryProver
	registryAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	// Since we can't directly access the RegistryProver struct, use the constructor
	prover := NewRegistryProver(mockEthClient, mockRPCClient, registryAddr)

	// Call the method being tested
	config, err := prover.GetL2Configuration(context.Background(), chainID)
	require.NoError(t, err)

	// Verify the results
	assert.Equal(t, configType, config.ConfigType)
	require.Len(t, config.Addresses, len(addresses))
	for i, addr := range addresses {
		testutil.RequireAddressEq(t, addr, config.Addresses[i])
	}
	require.Len(t, config.StorageSlots, len(slots))
	for i, slot := range slots {
		assert.Equal(t, slot.Uint64(), config.StorageSlots[i])
	}
}

func TestRegistryProver_GetL1BlockHashOracle(t *testing.T) {
	// Create test data
	chainID := uint64(10) // Optimism Chain ID
	oracleAddr := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")

	// Get the Registry ABI for testing
	registryABI, err := getRegistryABI()
	require.NoError(t, err)

	// Print all method signatures for debugging
	for name, method := range registryABI.Methods {
		t.Logf("Method: %s, Sig: 0x%x", name, method.ID)
	}

	// Create mock eth client that implements our IEthClient
	mockEthClient := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the right contract
			testutil.RequireAddressEq(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"), *msg.To)

			// Determine which method is being called by looking at the first 4 bytes of the calldata
			methodSig := msg.Data[:4]
			methodSigHex := hexutil.Encode(methodSig)
			t.Logf("Called method with signature: %s", methodSigHex)

			// Get method ID from the registry ABI
			getL1BlockHashOracleMethodID := registryABI.Methods["getL1BlockHashOracle"].ID

			if methodSigHex == hexutil.Encode(getL1BlockHashOracleMethodID) { // getL1BlockHashOracle(uint256)
				// Correctly pack the address according to the method output signature
				return registryABI.Methods["getL1BlockHashOracle"].Outputs.Pack(oracleAddr)
			}

			t.Fatalf("unexpected method signature: %s", methodSigHex)
			return nil, nil
		},
	}

	// Create mock RPC client that implements our IRPCClient
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			return nil
		},
	}

	// Create the RegistryProver
	registryAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	// Since we can't directly access the RegistryProver struct, use the constructor
	prover := NewRegistryProver(mockEthClient, mockRPCClient, registryAddr)

	// Call the method being tested
	result, err := prover.GetL1BlockHashOracle(context.Background(), chainID)
	require.NoError(t, err)

	// Verify the results
	testutil.RequireAddressEq(t, oracleAddr, result)
}

func TestRegistryProver_GetL2ConfigurationForUpdate(t *testing.T) {
	// Create test data
	chainID := uint64(42161) // Arbitrum One Chain ID
	addresses := []common.Address{
		common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
	}
	slots := []*big.Int{
		big.NewInt(1),
		big.NewInt(2),
	}
	configType := uint8(1) // OPStackBedrock
	proverAddr := common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432")
	versionNumber := big.NewInt(10)
	finalityDelaySeconds := big.NewInt(37800)

	// Get the Registry ABI for testing
	registryABI, err := getRegistryABI()
	require.NoError(t, err)

	// Create mock eth client that implements our IEthClient
	mockEthClient := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the right contract
			testutil.RequireAddressEq(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"), *msg.To)

			// Determine which method is being called by looking at the first 4 bytes of the calldata
			methodSig := msg.Data[:4]
			methodSigHex := hexutil.Encode(methodSig)
			t.Logf("Called method with signature: %s", methodSigHex)

			// Get method IDs from the registry ABI
			getL2ConfigTypeMethodID := registryABI.Methods["getL2ConfigType"].ID
			getL2ConfigAddressesMethodID := registryABI.Methods["getL2ConfigAddresses"].ID
			getL2ConfigStorageSlotsMethodID := registryABI.Methods["getL2ConfigStorageSlots"].ID
			l2ChainConfigurationsMethodID := registryABI.Methods["l2ChainConfigurations"].ID

			switch methodSigHex {
			case hexutil.Encode(getL2ConfigTypeMethodID): // getL2ConfigType(uint256)
				// Pack the method output with the uint8
				outputs := []interface{}{configType}
				packed, err := registryABI.Methods["getL2ConfigType"].Outputs.Pack(outputs...)
				require.NoError(t, err)
				return packed, nil
			case hexutil.Encode(getL2ConfigAddressesMethodID): // getL2ConfigAddresses(uint256)
				// Pack the address array
				return registryABI.Methods["getL2ConfigAddresses"].Outputs.Pack(addresses)
			case hexutil.Encode(getL2ConfigStorageSlotsMethodID): // getL2ConfigStorageSlots(uint256)
				// Pack the big int array
				return registryABI.Methods["getL2ConfigStorageSlots"].Outputs.Pack(slots)
			case hexutil.Encode(l2ChainConfigurationsMethodID): // l2ChainConfigurations(uint256)
				// This returns the struct from the mapping
				// We need to manually create a response that matches the expected ABI
				// The response for prover, versionNumber, finalityDelaySeconds, l2Type
				data := make([]byte, 128)                                                // 4 fields * 32 bytes
				copy(data[12:32], proverAddr.Bytes())                                    // prover address (padded to 32 bytes)
				copy(data[32:64], common.LeftPadBytes(versionNumber.Bytes(), 32))        // version number
				copy(data[64:96], common.LeftPadBytes(finalityDelaySeconds.Bytes(), 32)) // finality delay
				data[127] = configType                                                   // l2type uint8 (just the last byte)
				return data, nil
			default:
				t.Fatalf("unexpected method signature: %s", methodSigHex)
				return nil, nil
			}
		},
	}

	// Create mock RPC client
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			return nil
		},
	}

	// Create the RegistryProver
	registryAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	prover := NewRegistryProver(mockEthClient, mockRPCClient, registryAddr)

	// Call the method being tested
	config, err := prover.GetL2ConfigurationForUpdate(context.Background(), chainID)
	require.NoError(t, err)

	// Verify the results
	testutil.RequireAddressEq(t, proverAddr, config.Prover)
	require.Len(t, config.Addresses, len(addresses))
	for i, addr := range addresses {
		testutil.RequireAddressEq(t, addr, config.Addresses[i])
	}
	require.Len(t, config.StorageSlots, len(slots))
	for i, slot := range slots {
		assert.Equal(t, slot.String(), config.StorageSlots[i].String())
	}
	assert.Equal(t, versionNumber.String(), config.VersionNumber.String())
	assert.Equal(t, finalityDelaySeconds.String(), config.FinalityDelaySeconds.String())
	assert.Equal(t, types.OPStackBedrock, config.L2Type)
}

func TestRegistryProver_GetRegistryStorageProof(t *testing.T) {
	// Create test data
	chainID := uint64(42161) // Arbitrum One Chain ID
	registryAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	// Create test storage proof data
	mockAccountProof := []string{"0xproof1", "0xproof2"}
	mockStorageProofEntry := types.StorageProofEntry{
		Key:   common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		Value: (*hexutil.Big)(big.NewInt(0x123)),
		Proof: []string{"0xsproof1", "0xsproof2"},
	}

	// Create mock storage proof result that properly implements the type
	mockProofResult := types.StorageProofResult{
		Address:      registryAddr,
		Balance:      (*hexutil.Big)(big.NewInt(100)),
		CodeHash:     common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		Nonce:        (*hexutil.Uint64)(new(uint64)),
		StorageHash:  common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		AccountProof: mockAccountProof,
		StorageProof: []types.StorageProofEntry{mockStorageProofEntry},
	}
	*mockProofResult.Nonce = 1

	// Create mock RPC client that implements our IRPCClient
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			if method == "eth_getProof" {
				// Check arguments
				assert.Equal(t, "eth_getProof", method)
				assert.Len(t, args, 3) // address, slots array, block tag

				// Verify registry address matches
				addrArg, ok := args[0].(common.Address)
				require.True(t, ok, "Address argument should be common.Address, not %T", args[0])
				testutil.RequireAddressEq(t, registryAddr, addrArg)

				// Compute the expected slot hash just like in the implementation
				chainIDBytes := common.LeftPadBytes(big.NewInt(int64(chainID)).Bytes(), 32)
				mappingSlot := common.LeftPadBytes(big.NewInt(2).Bytes(), 32)
				slotPreimage := append(chainIDBytes, mappingSlot...)
				expectedSlotHash := common.BytesToHash(crypto.Keccak256(slotPreimage)).Hex()

				// Verify we're requesting a proof for the correct storage slot
				slotHashes, ok := args[1].([]string)
				require.True(t, ok)
				assert.Len(t, slotHashes, 1)
				assert.Equal(t, strings.ToLower(expectedSlotHash), strings.ToLower(slotHashes[0]))

				assert.Equal(t, "latest", args[2])

				// Set result by directly copying the mock result to the result pointer
				proofResult, ok := result.(*types.StorageProofResult)
				if !ok {
					return fmt.Errorf("result is not of type *StorageProofResult")
				}

				*proofResult = mockProofResult
				return nil
			}
			return nil
		},
	}

	// Create the RegistryProver - we only need the RPC client for this test
	prover := NewRegistryProver(nil, mockRPCClient, registryAddr)

	// Call the method being tested
	storageProof, rlpEncodedAccount, accountProof, err := prover.GetRegistryStorageProof(context.Background(), chainID)

	// Verify the results
	require.NoError(t, err)

	// Verify storageProof
	require.Len(t, storageProof, len(mockProofResult.StorageProof[0].Proof))
	for i, proofStr := range mockProofResult.StorageProof[0].Proof {
		assert.Equal(t, common.FromHex(proofStr), storageProof[i])
	}

	// Verify accountProof
	require.Len(t, accountProof, len(mockProofResult.AccountProof))
	for i, proofStr := range mockProofResult.AccountProof {
		assert.Equal(t, common.FromHex(proofStr), accountProof[i])
	}

	// Verify rlpEncodedAccount by creating an expected Account struct and RLP encoding it
	expectedAccount := Account{
		Nonce:    uint64(*mockProofResult.Nonce),
		Balance:  mockProofResult.Balance.ToInt(),
		Root:     mockProofResult.StorageHash,
		CodeHash: mockProofResult.CodeHash.Bytes(),
	}
	expectedRlpEncodedAccount, err := rlp.EncodeToBytes(expectedAccount)
	require.NoError(t, err)

	// Check that the actual encoded account matches our expected encoded account
	assert.Equal(t, expectedRlpEncodedAccount, rlpEncodedAccount)
}

func TestRegistryProver_GenerateUpdateL2ConfigArgs(t *testing.T) {
	// Create test data
	chainID := uint64(42161) // Arbitrum One Chain ID
	registryAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")

	// Setup test values
	proverAddr := common.HexToAddress("0x9876543210abcdef9876543210abcdef98765432")
	addresses := []common.Address{common.HexToAddress("0x1234")}
	slots := []*big.Int{big.NewInt(0x123)}
	versionNumber := big.NewInt(10)
	finalityDelaySeconds := big.NewInt(37800)
	configType := uint8(1) // OPStackBedrock

	// Get the Registry ABI for testing
	registryABI, err := getRegistryABI()
	require.NoError(t, err)

	// Create mock eth client for contract calls
	mockEthClient := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the right contract
			testutil.RequireAddressEq(t, registryAddr, *msg.To)

			// Determine which method is being called by looking at the first 4 bytes of the calldata
			methodSig := msg.Data[:4]
			methodSigHex := hexutil.Encode(methodSig)

			// Get method IDs from the registry ABI
			getL2ConfigTypeMethodID := registryABI.Methods["getL2ConfigType"].ID
			getL2ConfigAddressesMethodID := registryABI.Methods["getL2ConfigAddresses"].ID
			getL2ConfigStorageSlotsMethodID := registryABI.Methods["getL2ConfigStorageSlots"].ID
			l2ChainConfigurationsMethodID := registryABI.Methods["l2ChainConfigurations"].ID

			switch methodSigHex {
			case hexutil.Encode(getL2ConfigTypeMethodID): // getL2ConfigType(uint256)
				// Pack the method output with the uint8
				outputs := []interface{}{configType}
				packed, err := registryABI.Methods["getL2ConfigType"].Outputs.Pack(outputs...)
				require.NoError(t, err)
				return packed, nil

			case hexutil.Encode(getL2ConfigAddressesMethodID): // getL2ConfigAddresses(uint256)
				// Pack the address array
				return registryABI.Methods["getL2ConfigAddresses"].Outputs.Pack(addresses)

			case hexutil.Encode(getL2ConfigStorageSlotsMethodID): // getL2ConfigStorageSlots(uint256)
				// Pack the big int array
				return registryABI.Methods["getL2ConfigStorageSlots"].Outputs.Pack(slots)

			case hexutil.Encode(l2ChainConfigurationsMethodID): // l2ChainConfigurations(uint256)
				// Pack the struct data (prover, versionNumber, finalityDelaySeconds, l2Type)
				data := make([]byte, 128)                                                // 4 fields * 32 bytes
				copy(data[12:32], proverAddr.Bytes())                                    // prover address (padded to 32 bytes)
				copy(data[32:64], common.LeftPadBytes(versionNumber.Bytes(), 32))        // version number
				copy(data[64:96], common.LeftPadBytes(finalityDelaySeconds.Bytes(), 32)) // finality delay
				data[127] = configType                                                   // l2type uint8 (just the last byte)
				return data, nil

			default:
				t.Fatalf("unexpected method signature: %s", methodSigHex)
				return nil, nil
			}
		},
	}

	// Create mock storage proof data
	mockAccountProof := []string{"0xproof1", "0xproof2"}
	mockStorageProofEntry := types.StorageProofEntry{
		Key:   common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		Value: (*hexutil.Big)(big.NewInt(0x123)),
		Proof: []string{"0xsproof1", "0xsproof2"},
	}

	// Create mock storage proof result
	mockProofResult := types.StorageProofResult{
		Address:      registryAddr,
		Balance:      (*hexutil.Big)(big.NewInt(100)),
		CodeHash:     common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		Nonce:        (*hexutil.Uint64)(new(uint64)),
		StorageHash:  common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		AccountProof: mockAccountProof,
		StorageProof: []types.StorageProofEntry{mockStorageProofEntry},
	}
	*mockProofResult.Nonce = 1

	// Create mock RPC client for eth_getProof calls
	mockRPCClient := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			if method == "eth_getProof" {
				// Check arguments
				assert.Equal(t, "eth_getProof", method)
				assert.Len(t, args, 3) // address, slots array, block tag

				// The first argument should be the registry address
				addrArg, ok := args[0].(common.Address)
				require.True(t, ok, "Address argument should be common.Address, not %T", args[0])
				testutil.RequireAddressEq(t, registryAddr, addrArg)

				// Compute the expected slot hash just like in the implementation
				chainIDBytes := common.LeftPadBytes(big.NewInt(int64(chainID)).Bytes(), 32)
				mappingSlot := common.LeftPadBytes(big.NewInt(2).Bytes(), 32)
				slotPreimage := append(chainIDBytes, mappingSlot...)
				expectedSlotHash := common.BytesToHash(crypto.Keccak256(slotPreimage)).Hex()

				// Verify the slot hash is correct
				slotHashes := args[1].([]string)
				assert.Len(t, slotHashes, 1)
				assert.Equal(t, expectedSlotHash, slotHashes[0])

				assert.Equal(t, "latest", args[2])

				// Set result by copying mock proof result
				proofResult, ok := result.(*types.StorageProofResult)
				if !ok {
					return fmt.Errorf("result is not of type *types.StorageProofResult")
				}

				*proofResult = mockProofResult
				return nil
			}
			return nil
		},
	}

	// Create the RegistryProver
	prover := NewRegistryProver(mockEthClient, mockRPCClient, registryAddr)

	// Call the method being tested
	updateArgs, err := prover.GenerateUpdateL2ConfigArgs(context.Background(), chainID)
	require.NoError(t, err)

	// Verify the results
	testutil.RequireAddressEq(t, proverAddr, updateArgs.Config.Prover)
	require.Len(t, updateArgs.Config.Addresses, len(addresses))
	for i, addr := range addresses {
		testutil.RequireAddressEq(t, addr, updateArgs.Config.Addresses[i])
	}

	// Storage slots need to be compared one by one
	require.Equal(t, len(slots), len(updateArgs.Config.StorageSlots))
	for i, slot := range slots {
		assert.Equal(t, slot.String(), updateArgs.Config.StorageSlots[i].String())
	}

	assert.Equal(t, versionNumber.String(), updateArgs.Config.VersionNumber.String())
	assert.Equal(t, finalityDelaySeconds.String(), updateArgs.Config.FinalityDelaySeconds.String())
	assert.Equal(t, types.OPStackBedrock, updateArgs.Config.L2Type)

	// Verify proofs
	require.Len(t, updateArgs.L1StorageProof, len(mockProofResult.StorageProof[0].Proof))
	for i, proofStr := range mockProofResult.StorageProof[0].Proof {
		assert.Equal(t, common.FromHex(proofStr), updateArgs.L1StorageProof[i])
	}

	// Verify account proof
	require.Len(t, updateArgs.L1RegistryProof, len(mockProofResult.AccountProof))
	for i, proofStr := range mockProofResult.AccountProof {
		assert.Equal(t, common.FromHex(proofStr), updateArgs.L1RegistryProof[i])
	}

	// Verify RLP encoded account
	expectedAccount := Account{
		Nonce:    uint64(*mockProofResult.Nonce),
		Balance:  mockProofResult.Balance.ToInt(),
		Root:     mockProofResult.StorageHash,
		CodeHash: mockProofResult.CodeHash.Bytes(),
	}
	expectedRlpEncodedAccount, err := rlp.EncodeToBytes(expectedAccount)
	require.NoError(t, err)

	assert.Equal(t, expectedRlpEncodedAccount, updateArgs.RlpEncodedRegistryAccountData)
}
