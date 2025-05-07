package provers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	t "github.com/polymerdao/fallback_prover/types"
)

var _ INativeProver = &NativeProver{}

// NativeProver is responsible for encoding the calldata for the prove function
type NativeProver struct {
	abi abi.ABI
}

// NewNativeProver creates a new NativeProver
func NewNativeProver() (*NativeProver, error) {
	nativeProverABI, err := getNativeProverABI()
	if err != nil {
		return nil, fmt.Errorf("failed to get NativeProver ABI: %w", err)
	}

	return &NativeProver{
		abi: nativeProverABI,
	}, nil
}

// getNativeProverABI loads and parses the NativeProver ABI from file
func getNativeProverABI() (abi.ABI, error) {
	// Get the absolute path of the current file
	_, thisFile, _, _ := runtime.Caller(0)
	// Construct the path to the ABI file
	abiPath := filepath.Join(filepath.Dir(thisFile), "abis", "NativeProver.abi.json")

	// Read the ABI file
	abiFile, err := os.Open(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to open NativeProver ABI file: %w", err)
	}
	defer abiFile.Close()

	abiBytes, err := io.ReadAll(abiFile)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read NativeProver ABI file: %w", err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse NativeProver ABI: %w", err)
	}

	return parsedABI, nil
}

// EncodeProveNativeCalldata encodes the parameters for the NativeProver.proveNative() function call
func (np *NativeProver) EncodeProveNativeCalldata(
	updateArgs t.UpdateL2ConfigArgs,
	proveArgs t.ProveScalarArgs,
	rlpEncodedL1Header []byte,
	rlpEncodedL2Header []byte,
	settledStateProof []byte,
	l2StorageProof [][]byte,
	rlpEncodedContractAccount []byte,
	l2AccountProof [][]byte,
) ([]byte, error) {
	return np.abi.Pack(
		"proveNative",
		updateArgs,
		proveArgs,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
		settledStateProof,
		l2StorageProof,
		rlpEncodedContractAccount,
		l2AccountProof,
	)
}

// EncodeProveL1NativeCalldata encodes the parameters for the NativeProver.proveL1Native() function call
func (np *NativeProver) EncodeProveL1NativeCalldata(
	proveArgs t.ProveL1ScalarArgs,
	rlpEncodedL1Header []byte,
	l1StorageProof [][]byte,
	rlpEncodedContractAccount []byte,
	l1AccountProof [][]byte,
) ([]byte, error) {
	return np.abi.Pack(
		"proveL1Native",
		proveArgs,
		rlpEncodedL1Header,
		l1StorageProof,
		rlpEncodedContractAccount,
		l1AccountProof,
	)
}

// GetABI returns the ABI for the NativeProver
// This is mainly used for testing purposes
func (np *NativeProver) GetABI() abi.ABI {
	return np.abi
}
