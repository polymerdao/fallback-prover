package provers

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	t "github.com/polymerdao/fallback_prover/types"
)

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

// EncodeProveCalldata encodes the parameters for the NativeProver.prove() function call
func (np *NativeProver) EncodeProveCalldata(
	chainID *big.Int,
	contractAddr common.Address,
	storageSlot common.Hash,
	storageValue common.Hash,
	rlpEncodedL1Header []byte,
	rlpEncodedL2Header []byte,
	l2WorldStateRoot common.Hash,
	settledStateProof []byte,
	l2StorageProof [][]byte,
	rlpEncodedContractAccount []byte,
	l2AccountProof [][]byte,
) ([]byte, error) {
	// Create the ProveScalarArgs struct
	proveArgs := t.ProveScalarArgs{
		ChainID:          chainID,
		ContractAddr:     contractAddr,
		StorageSlot:      storageSlot,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2WorldStateRoot,
	}

	// Pack the arguments for the prove function
	return np.abi.Pack(
		"prove",
		proveArgs,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
		settledStateProof,
		l2StorageProof,
		rlpEncodedContractAccount,
		l2AccountProof,
	)
}

// EncodeUpdateAndProveCalldata encodes the parameters for the NativeProver.updateAndProve() function call
func (np *NativeProver) EncodeUpdateAndProveCalldata(
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
		"updateAndProve",
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

// EncodeConfigureAndProveCalldata encodes the parameters for the NativeProver.configureAndProve() function call
func (np *NativeProver) EncodeConfigureAndProveCalldata(
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
		"configureAndProve",
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

// EncodeProveL1Calldata encodes the parameters for the NativeProver.proveL1() function call
func (np *NativeProver) EncodeProveL1Calldata(
	proveArgs t.ProveL1ScalarArgs,
	rlpEncodedL1Header []byte,
	l1StorageProof [][]byte,
	rlpEncodedContractAccount []byte,
	l1AccountProof [][]byte,
) ([]byte, error) {
	return np.abi.Pack(
		"proveL1",
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
