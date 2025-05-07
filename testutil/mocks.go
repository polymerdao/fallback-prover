package testutil

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// MockEthClient is a mock implementation of the IEthClient for testing
type MockEthClient struct {
	CallContractFunc  func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	BlockByHashFunc   func(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumberFunc func(ctx context.Context, number *big.Int) (*types.Block, error)
}

func (m *MockEthClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if m.CallContractFunc != nil {
		return m.CallContractFunc(ctx, msg, blockNumber)
	}
	return nil, nil
}

func (m *MockEthClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	if m.BlockByHashFunc != nil {
		return m.BlockByHashFunc(ctx, hash)
	}
	return nil, nil
}

func (m *MockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if m.BlockByNumberFunc != nil {
		return m.BlockByNumberFunc(ctx, number)
	}
	return nil, nil
}

// MockRPCClient is a mock implementation of the IRPCClient for testing
type MockRPCClient struct {
	CallContextFunc      func(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContextFunc func(ctx context.Context, b []rpc.BatchElem) error
}

func (m *MockRPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if m.CallContextFunc != nil {
		return m.CallContextFunc(ctx, result, method, args...)
	}
	return nil
}

func (m *MockRPCClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	if m.BatchCallContextFunc != nil {
		return m.BatchCallContextFunc(ctx, b)
	}
	return nil
}

// CreateTestHeader creates a test block header
func CreateTestHeader(t *testing.T) *types.Header {
	return &types.Header{
		ParentHash:  common.HexToHash("0x123456"),
		UncleHash:   common.HexToHash("0x789abc"),
		Coinbase:    common.HexToAddress("0xdef123"),
		Root:        common.HexToHash("0x456789"),
		TxHash:      common.HexToHash("0xabcdef"),
		ReceiptHash: common.HexToHash("0x12345678"),
		Bloom:       types.BytesToBloom(make([]byte, 256)),
		Difficulty:  big.NewInt(1000),
		Number:      big.NewInt(12345),
		GasLimit:    uint64(1000000),
		GasUsed:     uint64(500000),
		Time:        uint64(1000000000),
		Extra:       []byte("test"),
		MixDigest:   common.HexToHash("0x87654321"),
	}
}

// CreateTestBlock creates a test block with the given header
func CreateTestBlock(t *testing.T, header *types.Header) *types.Block {
	return types.NewBlockWithHeader(header)
}

// MockStorageProofResult creates a mock storage proof result for testing
func MockStorageProofResult(t *testing.T, address common.Address, slot common.Hash, value *big.Int) map[string]interface{} {
	// Convert the address to a checksum address
	checksumAddr := address.Hex()

	// Create a storage proof entry for the given slot and value
	storageProof := []map[string]interface{}{
		{
			"key":   slot.Hex(),
			"value": "0x" + value.Text(16),
			"proof": []string{
				"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
	}

	// Create the full proof result
	return map[string]interface{}{
		"address":      checksumAddr,
		"accountProof": []string{"0xproof1", "0xproof2"},
		"balance":      "0x0",
		"codeHash":     "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"nonce":        "0x0",
		"storageHash":  "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"storageProof": storageProof,
	}
}

// RequireAddressEq compares common.Address values
func RequireAddressEq(t *testing.T, expected, actual common.Address) {
	require.Equal(t, expected.Hex(), actual.Hex())
}

// RequireHashEq compares common.Hash values
func RequireHashEq(t *testing.T, expected, actual common.Hash) {
	require.Equal(t, expected.Hex(), actual.Hex())
}
