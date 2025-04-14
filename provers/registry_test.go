package provers

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/polymerdao/fallback_prover/testutil"
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
			methodSig := hexutil.Encode(msg.Data[:4])
			t.Logf("Called method with signature: %s", methodSig)

			if methodSig == "0x7fc0ad31" { // getL1BlockHashOracle(uint256)
				// Correctly pack the address according to the method output signature
				return registryABI.Methods["getL1BlockHashOracle"].Outputs.Pack(oracleAddr)
			}

			t.Fatalf("unexpected method signature: %s", methodSig)
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
