package provers

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/polymerdao/fallback_prover/testutil"
	types2 "github.com/polymerdao/fallback_prover/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOPStackCannonProver_GenerateSettledStateProof(t *testing.T) {
	// Create test data
	disputeGameFactoryAddr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	disputeGameAddr := common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	gameCount := big.NewInt(5)
	gameIndex := big.NewInt(4) // The most recent game
	rootClaim := common.HexToHash("0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba")
	gameStatus := uint8(2) // Some status value
	messagePasserRoot := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")
	faultDisputeGameStateRoot := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	l1StateRoot := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	// Create a test header and block
	l2Header := testutil.CreateTestHeader(t)
	l2Block := testutil.CreateTestBlock(t, l2Header)

	// Create L2 config
	config := &types2.L2ConfigInfo{
		ConfigType: "OPStackCannon",
		Addresses: []common.Address{
			disputeGameFactoryAddr,
		},
		StorageSlots: []uint64{
			0x123, // DisputeGameFactory list slot
			0x456, // FaultDisputeGame rootClaim slot
			0x789, // FaultDisputeGame status slot
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
			// Check that we're calling one of the expected contracts
			if msg.To.Hex() == disputeGameFactoryAddr.Hex() {
				// Determine which method is being called by looking at the method signature
				methodSig := msg.Data[:4]

				// gameCount signature: 0x9ac1d59f
				if string(methodSig) == string(hexutil.MustDecode("0x9ac1d59f")) {
					// Fix: directly return the packed uint256
					return common.LeftPadBytes(gameCount.Bytes(), 32), nil
				}

				// gameAtIndex signature: 0xaf640d0f
				if string(methodSig) == string(hexutil.MustDecode("0xaf640d0f")) {
					// Fix: directly return the address as a 32-byte value
					return common.LeftPadBytes(disputeGameAddr.Bytes(), 32), nil
				}
			} else if msg.To.Hex() == disputeGameAddr.Hex() {
				// Determine which method is being called by looking at the method signature
				methodSig := msg.Data[:4]

				// rootClaim signature: 0x74a1e471
				if string(methodSig) == string(hexutil.MustDecode("0x74a1e471")) {
					// Fix: directly return the hash as bytes
					return rootClaim.Bytes(), nil
				}

				// status signature: 0x200d2ed2
				if string(methodSig) == string(hexutil.MustDecode("0x200d2ed2")) {
					// Fix: directly return the status as a byte
					statusBytes := []byte{gameStatus}
					return common.RightPadBytes(statusBytes, 32), nil
				}
			}

			return nil, nil
		},
	}

	// Create mock RPC clients that handle eth_getProof
	mockL1RPC := &testutil.MockRPCClient{
		CallContextFunc: func(ctx context.Context, result interface{}, method string, args ...interface{}) error {
			if method == "eth_getProof" {
				// We need to handle proofs for both the dispute game factory and the fault dispute game
				address := args[0].(string)

				if address == disputeGameFactoryAddr.Hex() {
					// Mock a dispute game factory proof
					mockProof := testutil.MockStorageProofResult(
						t,
						disputeGameFactoryAddr,
						common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000123"),
						gameIndex,
					)

					// Marshal to JSON and unmarshal into the result
					mockProofJSON, err := json.Marshal(mockProof)
					require.NoError(t, err)
					return json.Unmarshal(mockProofJSON, result)
				} else if address == disputeGameAddr.Hex() {
					// Mock a fault dispute game proof
					mockProof := map[string]interface{}{
						"address":      disputeGameAddr.Hex(),
						"accountProof": []string{"0xproof1", "0xproof2"},
						"balance":      "0x0",
						"codeHash":     "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
						"nonce":        "0x0",
						"storageHash":  faultDisputeGameStateRoot.Hex(),
						"storageProof": []map[string]interface{}{
							{
								"key":   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000456").Hex(),
								"value": rootClaim.Hex(),
								"proof": []string{"0xproof1", "0xproof2"},
							},
							{
								"key":   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000789").Hex(),
								"value": "0x0000000000000000000000000000000000000000000000000000000000000002", // Game status
								"proof": []string{"0xproof1", "0xproof2"},
							},
						},
					}

					// Marshal to JSON and unmarshal into the result
					mockProofJSON, err := json.Marshal(mockProof)
					require.NoError(t, err)
					return json.Unmarshal(mockProofJSON, result)
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
	}

	// Set up RLP encoded L2 header for test to check if it matches the result
	expectedRlpEncodedL2Header, err := rlp.EncodeToBytes(l2Header)
	require.NoError(t, err)

	// Create the OPStackCannonProver
	prover, err := NewOPStackCannonProver(mockL1Client, mockL1RPC, mockL2Client, mockL2RPC)
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
	assert.Equal(t, expectedRlpEncodedL2Header, rlpEncodedL2Header)
}
