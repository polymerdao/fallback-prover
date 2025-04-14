package provers

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/polymerdao/fallback_prover/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// RegistryProver handles interactions with the Registry contract on L1
type RegistryProver struct {
	l1Client     IEthClient
	l1RPC        IRPCClient
	registryAddr common.Address
	abi          abi.ABI
}

// NewRegistryProver creates a new RegistryProver
func NewRegistryProver(l1Client IEthClient, l1RPC IRPCClient, registryAddr common.Address) *RegistryProver {
	registryABI, err := getRegistryABI()
	if err != nil {
		panic(fmt.Sprintf("failed to load Registry ABI: %v", err))
	}

	return &RegistryProver{
		l1Client:     l1Client,
		l1RPC:        l1RPC,
		registryAddr: registryAddr,
		abi:          registryABI,
	}
}

// getRegistryABI loads and parses the Registry ABI from file
func getRegistryABI() (abi.ABI, error) {
	// Get the absolute path of the current file
	_, thisFile, _, _ := runtime.Caller(0)
	// Construct the path to the ABI file
	abiPath := filepath.Join(filepath.Dir(thisFile), "abis", "Registry.abi.json")

	// Read the ABI file
	abiFile, err := os.Open(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to open Registry ABI file: %w", err)
	}
	defer abiFile.Close()

	abiBytes, err := io.ReadAll(abiFile)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read Registry ABI file: %w", err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse Registry ABI: %w", err)
	}

	return parsedABI, nil
}

// convertTypeToString converts the L2 config enum value to a string
func convertTypeToString(typeValue uint8) string {
	switch typeValue {
	case 0:
		return "Unknown"
	case 1:
		return "OPStackBedrock"
	case 2:
		return "OPStackCannon"
	case 3:
		return "Arbitrum"
	default:
		return fmt.Sprintf("Undefined:%d", typeValue)
	}
}

// GetL2Configuration fetches the L2 configuration for a given chain ID
func (r *RegistryProver) GetL2Configuration(ctx context.Context, chainID uint64) (*types.L2ConfigInfo, error) {
	// 1. Query for the L2 config type
	configTypeData, err := r.abi.Pack("getL2ConfigType", big.NewInt(int64(chainID)))
	if err != nil {
		return nil, fmt.Errorf("failed to pack getL2ConfigType: %w", err)
	}

	configTypeResult, err := r.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &r.registryAddr,
		Data: configTypeData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call getL2ConfigType: %w", err)
	}

	// Unpack the config type (which is actually a uint8 enum in the contract)
	var configTypeUint8 uint8
	if err := r.abi.UnpackIntoInterface(&configTypeUint8, "getL2ConfigType", configTypeResult); err != nil {
		return nil, fmt.Errorf("failed to unpack config type: %w", err)
	}

	// Convert uint8 to string representation
	configType := convertTypeToString(configTypeUint8)

	// 2. Query for addresses
	addressesData, err := r.abi.Pack("getL2ConfigAddresses", big.NewInt(int64(chainID)))
	if err != nil {
		return nil, fmt.Errorf("failed to pack getL2ConfigAddresses: %w", err)
	}

	addressesResult, err := r.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &r.registryAddr,
		Data: addressesData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call getL2ConfigAddresses: %w", err)
	}

	// Unpack the addresses
	var addresses []common.Address
	if err := r.abi.UnpackIntoInterface(&addresses, "getL2ConfigAddresses", addressesResult); err != nil {
		return nil, fmt.Errorf("failed to unpack addresses: %w", err)
	}

	// 3. Query for storage slots
	slotsData, err := r.abi.Pack("getL2ConfigStorageSlots", big.NewInt(int64(chainID)))
	if err != nil {
		return nil, fmt.Errorf("failed to pack getL2ConfigStorageSlots: %w", err)
	}

	slotsResult, err := r.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &r.registryAddr,
		Data: slotsData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call getL2ConfigStorageSlots: %w", err)
	}

	// Unpack the storage slots
	var slots []*big.Int
	if err := r.abi.UnpackIntoInterface(&slots, "getL2ConfigStorageSlots", slotsResult); err != nil {
		return nil, fmt.Errorf("failed to unpack storage slots: %w", err)
	}

	// Convert *big.Int slots to uint64 array
	var uintSlots []uint64
	for _, slot := range slots {
		uintSlots = append(uintSlots, slot.Uint64())
	}

	return &types.L2ConfigInfo{
		ConfigType:   configType,
		Addresses:    addresses,
		StorageSlots: uintSlots,
	}, nil
}

// GetL1BlockHashOracle fetches the L1 block hash oracle address for a given L2 chain ID
func (r *RegistryProver) GetL1BlockHashOracle(ctx context.Context, chainID uint64) (common.Address, error) {
	// Query for the L1 block hash oracle
	oracleData, err := r.abi.Pack("getL1BlockHashOracle", big.NewInt(int64(chainID)))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack getL1BlockHashOracle: %w", err)
	}

	oracleResult, err := r.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &r.registryAddr,
		Data: oracleData,
	}, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to call getL1BlockHashOracle: %w", err)
	}

	// Unpack the oracle address
	var oracleAddr common.Address
	if err := r.abi.UnpackIntoInterface(&oracleAddr, "getL1BlockHashOracle", oracleResult); err != nil {
		return common.Address{}, fmt.Errorf("failed to unpack L1 block hash oracle address: %w", err)
	}

	return oracleAddr, nil
}
