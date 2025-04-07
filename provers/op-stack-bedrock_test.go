package provers

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/polymerdao/fallback_prover/testutil"
	types2 "github.com/polymerdao/fallback_prover/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOPStackBedrockProver_GenerateSettledStateProof(t *testing.T) {
	// Create a temporary directory for the ABI file
	tempDir, err := os.MkdirTemp("", "opstack-bedrock-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create the abis directory
	abisDir := filepath.Join(tempDir, "abis")
	err = os.Mkdir(abisDir, 0755)
	require.NoError(t, err)

	// Create the OPStackBedrockProver ABI file
	abiContent := `[
		{
			"inputs": [
				{
					"internalType": "bytes32",
					"name": "_messagePasserStorageRoot",
					"type": "bytes32"
				},
				{
					"internalType": "uint256",
					"name": "_outputIndex",
					"type": "uint256"
				},
				{
					"internalType": "bytes[]",
					"name": "_outputRootProof",
					"type": "bytes[]"
				},
				{
					"internalType": "bytes",
					"name": "_outputOracleData",
					"type": "bytes"
				},
				{
					"internalType": "bytes[]",
					"name": "_outputOracleProof",
					"type": "bytes[]"
				}
			],
			"name": "proveSettledState",
			"outputs": [
				{
					"internalType": "bool",
					"name": "",
					"type": "bool"
				}
			],
			"stateMutability": "view",
			"type": "function"
		}
	]`
	err = os.WriteFile(filepath.Join(abisDir, "OPStackBedrockProver.abi.json"), []byte(abiContent), 0644)
	require.NoError(t, err)

	// Parse the L2OutputOracle ABI
	l2OutputOracleABI, err := getL2OutputOracleABI()
	require.NoError(t, err)

	// Create test data
	l2OutputOracleAddr := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	outputIndex := big.NewInt(123)
	l1StateRoot := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	messagePasserRoot := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")

	// Create a test header and block
	l2Header := testutil.CreateTestHeader(t)
	l2Block := testutil.CreateTestBlock(t, l2Header)

	// Create L2 config
	config := &types2.L2ConfigInfo{
		ConfigType: "OPStackBedrock",
		Addresses: []common.Address{
			l2OutputOracleAddr,
		},
		StorageSlots: []uint64{
			0x123, // Some storage slot value
		},
	}

	// Create mock L2 client
	mockL2Client := &testutil.MockEthClient{
		BlockByNumberFunc: func(ctx context.Context, number *big.Int) (*types.Block, error) {
			return l2Block, nil
		},
	}

	// Create mock L1 client
	mockL1Client := &testutil.MockEthClient{
		CallContractFunc: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			// Check that we're calling the right contract
			require.Equal(t, l2OutputOracleAddr.Hex(), msg.To.Hex())

			// Determine which method is being called by looking at the method signature
			methodSig := msg.Data[:4]

			// getL2OutputIndexAfter signature: 0x7f006420
			if string(methodSig) == string(hexutil.MustDecode("0x7f006420")) {
				packedData, err := l2OutputOracleABI.Pack("getL2OutputIndexAfter", outputIndex)
				require.NoError(t, err)
				return packedData, nil
			}

			// getL2Output signature: 0xa25ae557
			if string(methodSig) == string(hexutil.MustDecode("0xa25ae557")) {
				// We need to construct a tuple with (outputRoot, timestamp, l2BlockNumber)
				outputProposal := struct {
					OutputRoot    common.Hash
					Timestamp     *big.Int
					L2BlockNumber *big.Int
				}{
					OutputRoot:    common.HexToHash("0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba"),
					Timestamp:     big.NewInt(1000000000),
					L2BlockNumber: big.NewInt(12345),
				}
				packedData, err := l2OutputOracleABI.Pack("getL2Output", outputProposal)
				require.NoError(t, err)
				return packedData, nil
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
	}

	// Create the OPStackBedrockProver
	prover, err := NewOPStackBedrockProver(mockL1Client, mockL1RPC, mockL2Client, mockL2RPC)
	require.NoError(t, err)

	// Call the method being tested
	settledStateProof, l2StateRoot, rlpEncodedL2Header, err := prover.GenerateSettledStateProof(
		context.Background(),
		config,
		l1StateRoot,
	)
	require.NoError(t, err)

	// Verify the results
	assert.NotNil(t, settledStateProof)
	assert.Equal(t, l2Header.Root.Hex(), l2StateRoot.Hex())
	assert.NotNil(t, rlpEncodedL2Header)
}
