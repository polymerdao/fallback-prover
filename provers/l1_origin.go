package provers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// L1OriginProver handles proving L1 origins for L2 chains
type L1OriginProver struct {
	l1Client    IEthClient
	l2Client    IEthClient
	l1OracleABI abi.ABI
}

// NewL1OriginProver creates a new L1OriginProver
func NewL1OriginProver(l1Client IEthClient, l2Client IEthClient) *L1OriginProver {
	il1BlockABI, err := getIL1BlockABI()
	if err != nil {
		panic(fmt.Sprintf("failed to load IL1Block ABI: %v", err))
	}

	return &L1OriginProver{
		l1Client:    l1Client,
		l2Client:    l2Client,
		l1OracleABI: il1BlockABI,
	}
}

// getIL1BlockABI loads and parses the IL1Block ABI from file
func getIL1BlockABI() (abi.ABI, error) {
	// Get the absolute path of the current file
	_, thisFile, _, _ := runtime.Caller(0)
	// Construct the path to the ABI file
	abiPath := filepath.Join(filepath.Dir(thisFile), "abis", "IL1Block.abi.json")

	// Read the ABI file
	abiFile, err := os.Open(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to open IL1Block ABI file: %w", err)
	}
	defer abiFile.Close()

	abiBytes, err := io.ReadAll(abiFile)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read IL1Block ABI file: %w", err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse IL1Block ABI: %w", err)
	}

	return parsedABI, nil
}

func (l *L1OriginProver) GetL1OriginHash(ctx context.Context, l1OracleAddress common.Address) (common.Hash, error) {
	data, err := l.l1OracleABI.Pack("hash")
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to pack hash() call: %w", err)
	}

	result, err := l.l2Client.CallContract(ctx, ethereum.CallMsg{
		To:   &l1OracleAddress,
		Data: data,
	}, nil) // Use latest block
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to call hash() on L1 oracle: %w", err)
	}

	// Unpack the result to get the L1 header hash
	if len(result) != 32 {
		return common.Hash{}, fmt.Errorf("unexpected result length: %d", len(result))
	}

	return common.BytesToHash(result), nil
}

func (l *L1OriginProver) GetL1Origin(ctx context.Context, l1OriginHash common.Hash) ([]byte, *types.Header, error) {
	l1Block, err := l.l1Client.BlockByHash(ctx, l1OriginHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get L1 block by hash: %w", err)
	}

	encodedHeader, err := rlp.EncodeToBytes(l1Block.Header())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to RLP encode L1 header: %w", err)
	}

	return encodedHeader, l1Block.Header(), nil
}
