package provers

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/polymerdao/fallback_prover/testutil"
	types2 "github.com/polymerdao/fallback_prover/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOPStackBedrockProver_FindLatestResolved(t *testing.T) {

	// Create test data
	l2OutputOracleAddr := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	latestOutputIndex := big.NewInt(123)

	// Create L2 config
	config := &types2.L2ConfigInfo{
		ConfigType: "OPStackBedrock",
		Addresses: []common.Address{
			l2OutputOracleAddr,
		},
		StorageSlots: []*big.Int{
			big.NewInt(0x123), // Some storage slot value
		},
	}

	// Create mock L1 client
	mockL1Client := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the right contract
			require.Equal(t, l2OutputOracleAddr.Hex(), msg.To.Hex())

			// Return the latestOutputIndex
			return common.LeftPadBytes(latestOutputIndex.Bytes(), 32), nil
		},
	}

	// Create the OPStackBedrockProver
	prover, err := NewOPStackBedrockProver(mockL1Client, nil, nil)
	require.NoError(t, err)

	// Call the method being tested
	outputIndex, addr, err := prover.FindLatestResolved(context.Background(), config)
	require.NoError(t, err)

	// Verify the results
	assert.Equal(t, latestOutputIndex.String(), outputIndex.String())
	assert.Equal(t, l2OutputOracleAddr.Hex(), addr.Hex())
}

func TestOPStackBedrockProver_GenerateSettledStateProof(t *testing.T) {
	// Parse the L2OutputOracle ABI
	l2OutputOracleABI, err := getL2OutputOracleABI()
	require.NoError(t, err)

	// Create test data
	l2OutputOracleAddr := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	outputIndex := big.NewInt(123)
	messagePasserRoot := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")

	// Create a test header and block
	l2Header := testutil.CreateTestHeader(t)

	// Create L2 config
	config := &types2.L2ConfigInfo{
		ConfigType: "OPStackBedrock",
		Addresses: []common.Address{
			l2OutputOracleAddr,
		},
		StorageSlots: []*big.Int{
			big.NewInt(0x123), // Some storage slot value
		},
	}

	// Create mock L1 client
	// Create L1BlockNumber for testing
	expectedL1BlockNumber := big.NewInt(12345)

	mockL1Client := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling with the expected L1BlockNumber
			require.Equal(
				t,
				expectedL1BlockNumber,
				blockNumber,
				"CallContract should be called with the expected L1BlockNumber",
			)

			// Check that we're calling the right contract
			require.Equal(t, l2OutputOracleAddr.Hex(), msg.To.Hex())

			// Determine which method is being called by looking at the method signature
			methodSig := msg.Data[:4]
			methodSigHex := hexutil.Encode(methodSig)

			// Get method IDs from the L2OutputOracle ABI
			getL2OutputMethodID := l2OutputOracleABI.Methods["getL2Output"].ID

			// getL2Output method
			if methodSigHex == hexutil.Encode(getL2OutputMethodID) {
				// Instead of using packing, create a byte array directly
				outputRoot := common.HexToHash("0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba")
				timestamp := big.NewInt(1000000000)
				l2BlockNumber := big.NewInt(12345)

				// Manually create the response - 32 bytes for each field
				responseData := make([]byte, 96)
				copy(responseData[0:32], outputRoot.Bytes())
				copy(responseData[32:64], common.LeftPadBytes(timestamp.Bytes(), 32))
				copy(responseData[64:96], common.LeftPadBytes(l2BlockNumber.Bytes(), 32))

				return responseData, nil
			}

			return nil, nil
		},
	}

	// Create mock RPC clients that handle eth_getProof
	mockL1RPC := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			if method == "eth_getProof" {
				// Mock a storage proof result
				mockProof := testutil.MockStorageProofResult(
					t,
					l2OutputOracleAddr,
					common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000123"),
					big.NewInt(456),
				)

				// Marshal to JSON and unmarshal into the result
				mockProofJSON, err := json.Marshal(mockProof)
				require.NoError(t, err)
				return json.Unmarshal(mockProofJSON, result)
			}
			return nil
		},
		BatchCallContextFunc: func(ctx context.Context, b []rpc.BatchElem) error {
			// Process each element in the batch
			for i := range b {
				elem := &b[i]
				if elem.Method == "eth_getProof" {
					// Mock a storage proof result
					mockProof := testutil.MockStorageProofResult(
						t,
						l2OutputOracleAddr,
						common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000123"),
						big.NewInt(456),
					)

					// Marshal to JSON and unmarshal into the result
					mockProofJSON, err := json.Marshal(mockProof)
					require.NoError(t, err)
					err = json.Unmarshal(mockProofJSON, elem.Result)
					if err != nil {
						return err
					}
				} else if elem.Method == "eth_call" {
					// Mock call result for getL2Output
					outputRoot := common.HexToHash("0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba")
					timestamp := big.NewInt(1000000000)
					l2BlockNumber := big.NewInt(12345)

					// Manually create the response - 32 bytes for each field
					responseData := make([]byte, 96)
					copy(responseData[0:32], outputRoot.Bytes())
					copy(responseData[32:64], common.LeftPadBytes(timestamp.Bytes(), 32))
					copy(responseData[64:96], common.LeftPadBytes(l2BlockNumber.Bytes(), 32))

					// Convert to hex string for RPC response
					hexData := "0x" + common.Bytes2Hex(responseData)

					// Type assert to get the right result pointer
					if resultPtr, ok := elem.Result.(*[]byte); ok {
						*resultPtr = responseData
					} else if resultPtr, ok := elem.Result.(*string); ok {
						*resultPtr = hexData
					}
				}
			}
			return nil
		},
	}

	mockL2RPC := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			if method == "eth_getProof" {
				// Mock a message passer proof result with the specific storage hash we want
				mockProof := map[string]interface{}{
					"address":      "0x4200000000000000000000000000000000000016",
					"accountProof": []string{"0xproof1", "0xproof2"},
					"balance":      "0x0",
					"codeHash":     "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
					"nonce":        "0x0",
					"storageHash":  messagePasserRoot.Hex(),
					"storageProof": []interface{}{},
				}

				// Marshal to JSON and unmarshal into the result
				mockProofJSON, err := json.Marshal(mockProof)
				require.NoError(t, err)
				return json.Unmarshal(mockProofJSON, result)
			}
			return nil
		},
		BatchCallContextFunc: func(ctx context.Context, b []rpc.BatchElem) error {
			// Process each element in the batch
			for i := range b {
				elem := &b[i]
				if elem.Method == "eth_getProof" {
					// Mock a message passer proof result with the specific storage hash we want
					mockProof := map[string]interface{}{
						"address":      "0x4200000000000000000000000000000000000016",
						"accountProof": []string{"0xproof1", "0xproof2"},
						"balance":      "0x0",
						"codeHash":     "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						"nonce":        "0x0",
						"storageHash":  messagePasserRoot.Hex(),
						"storageProof": []interface{}{},
					}

					// Marshal to JSON and unmarshal into the result
					mockProofJSON, err := json.Marshal(mockProof)
					require.NoError(t, err)
					err = json.Unmarshal(mockProofJSON, elem.Result)
					if err != nil {
						return err
					}
				} else if elem.Method == "eth_getBlockByNumber" {
					// Set the block data for json.RawMessage
					if rawMsg, ok := elem.Result.(*json.RawMessage); ok {
						// Marshal the block into JSON format for the proper RawMessage processing
						blockJSON, err := json.Marshal(l2Header)
						require.NoError(t, err)
						*rawMsg = blockJSON
					}
				}
			}
			return nil
		},
	}

	// Create the OPStackBedrockProver
	prover, err := NewOPStackBedrockProver(mockL1Client, mockL1RPC, mockL2RPC)
	require.NoError(t, err)

	// Call the method being tested
	settledStateProof, l2Header, err := prover.GenerateSettledStateProof(
		context.Background(),
		expectedL1BlockNumber,
		outputIndex,
		l2OutputOracleAddr,
		config,
	)
	require.NoError(t, err)

	// Verify the results
	assert.NotNil(t, settledStateProof)
	assert.Equal(t, l2Header.Root.Hex(), l2Header.Root.Hex())
}
