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

	t "github.com/polymerdao/fallback_prover/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
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
func (r *RegistryProver) GetL2Configuration(ctx context.Context, chainID uint64) (*t.L2ConfigInfo, error) {
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

	return &t.L2ConfigInfo{
		ConfigType:   configType,
		Addresses:    addresses,
		StorageSlots: slots,
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

// GetL2ConfigurationForUpdate retrieves the complete L2Configuration for generating update proofs
func (r *RegistryProver) GetL2ConfigurationForUpdate(ctx context.Context, chainID uint64) (*t.L2Configuration, error) {
	chainIDParam := big.NewInt(int64(chainID))

	// Get L2 config type (enum value)
	configTypeData, err := r.abi.Pack("getL2ConfigType", chainIDParam)
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

	var l2TypeEnum uint8
	if err := r.abi.UnpackIntoInterface(&l2TypeEnum, "getL2ConfigType", configTypeResult); err != nil {
		return nil, fmt.Errorf("failed to unpack L2 type: %w", err)
	}

	// Map enum value to our L2Type
	var l2Type t.L2Type
	switch l2TypeEnum {
	case 1: // OPStackBedrock (based on the contract enum)
		l2Type = t.OPStackBedrock
	case 2: // OPStackCannon (based on the contract enum)
		l2Type = t.OPStackCannon
	default:
		return nil, fmt.Errorf("unsupported L2 type enum value: %d", l2TypeEnum)
	}

	// Get addresses from the config
	addressesData, err := r.abi.Pack("getL2ConfigAddresses", chainIDParam)
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

	var addresses []common.Address
	if err := r.abi.UnpackIntoInterface(&addresses, "getL2ConfigAddresses", addressesResult); err != nil {
		return nil, fmt.Errorf("failed to unpack addresses: %w", err)
	}

	// Get storage slots from the config
	slotsData, err := r.abi.Pack("getL2ConfigStorageSlots", chainIDParam)
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

	var bigIntSlots []*big.Int
	if err := r.abi.UnpackIntoInterface(&bigIntSlots, "getL2ConfigStorageSlots", slotsResult); err != nil {
		return nil, fmt.Errorf("failed to unpack storage slots: %w", err)
	}

	// Get the l2ChainConfigurations mapping data for versionNumber and finalityDelaySeconds
	l2ConfigData, err := r.abi.Pack("l2ChainConfigurations", chainIDParam)
	if err != nil {
		return nil, fmt.Errorf("failed to pack l2ChainConfigurations: %w", err)
	}

	l2ConfigResult, err := r.l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &r.registryAddr,
		Data: l2ConfigData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call l2ChainConfigurations: %w", err)
	}

	// The return type of l2ChainConfigurations is a struct with:
	// address prover, uint256 versionNumber, uint256 finalityDelaySeconds, uint8 l2Type
	// We need prover, versionNumber and finalityDelaySeconds
	var prover common.Address
	var versionNumber *big.Int
	var finalityDelaySeconds *big.Int
	// We're using a simple unpacking approach here
	if len(l2ConfigResult) >= 128 { // 4 fields * 32 bytes
		prover = common.BytesToAddress(l2ConfigResult[12:32])               // Address is padded to 32 bytes
		versionNumber = new(big.Int).SetBytes(l2ConfigResult[32:64])        // Second field
		finalityDelaySeconds = new(big.Int).SetBytes(l2ConfigResult[64:96]) // Third field
	} else {
		return nil, fmt.Errorf("invalid response length from l2ChainConfigurations: %d", len(l2ConfigResult))
	}

	return &t.L2Configuration{
		Prover:               prover,
		Addresses:            addresses,
		StorageSlots:         bigIntSlots,
		VersionNumber:        versionNumber,
		FinalityDelaySeconds: finalityDelaySeconds,
		L2Type:               l2Type,
	}, nil
}

// GetRegistryStorageProof gets a storage proof for the registry contract
func (r *RegistryProver) GetRegistryStorageProof(
	ctx context.Context,
	chainID uint64,
	blockNum *big.Int,
) ([][]byte, []byte, [][]byte, error) {
	// Calculate the storage slot for l2ChainConfigurationHashMap[chainID]
	// In Solidity, the storage slot for mapping(uint256 => bytes32) at position X is keccak256(key . X)
	// where . is concatenation and X is the position (padded to 32 bytes)

	// In the Registry contract, l2ChainConfigurationHashMap is at slot 2
	chainIDBytes := common.LeftPadBytes(big.NewInt(int64(chainID)).Bytes(), 32)
	mappingSlot := common.LeftPadBytes(big.NewInt(2).Bytes(), 32)

	// Calculate the actual storage slot: keccak256(chainID + mappingSlot)
	slotPreimage := append(chainIDBytes, mappingSlot...)
	slotHash := common.BytesToHash(crypto.Keccak256(slotPreimage))

	// Use eth_getProof to generate the proof
	var result t.StorageProofResult
	err := r.l1RPC.CallContext(
		ctx,
		&result,
		"eth_getProof",
		r.registryAddr,
		[]string{slotHash.Hex()},
		toBlockNumArg(blockNum),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get storage proof from registry: %w", err)
	}

	// Convert account proof to bytes
	accountProof := make([][]byte, len(result.AccountProof))
	for i, p := range result.AccountProof {
		accountProof[i] = common.FromHex(p)
	}

	// Check if we have a storage proof
	if len(result.StorageProof) == 0 {
		return nil, nil, nil, fmt.Errorf("no storage proof found for L2 configuration in registry")
	}

	// Convert storage proof to bytes
	storageProof := make([][]byte, len(result.StorageProof[0].Proof))
	for i, p := range result.StorageProof[0].Proof {
		storageProof[i] = common.FromHex(p)
	}

	// Create an account object
	account := Account{
		Nonce:    uint64(*result.Nonce),
		Balance:  result.Balance.ToInt(),
		Root:     result.StorageHash,
		CodeHash: result.CodeHash.Bytes(),
	}

	// RLP encode the account
	rlpEncodedAccount, err := rlp.EncodeToBytes(account)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to RLP encode registry account: %w", err)
	}

	return storageProof, rlpEncodedAccount, accountProof, nil
}

// GenerateUpdateL2ConfigArgs builds a complete UpdateL2ConfigArgs structure
func (r *RegistryProver) GenerateUpdateL2ConfigArgs(
	ctx context.Context,
	chainID uint64,
	blockNumber *big.Int,
) (*t.UpdateL2ConfigArgs, error) {
	// Get the L2 configuration
	l2Config, err := r.GetL2ConfigurationForUpdate(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get L2 configuration: %w", err)
	}

	// Get the registry storage proof
	l1StorageProof, rlpEncodedRegistryData, l1RegistryProof, err := r.GetRegistryStorageProof(
		ctx,
		chainID,
		blockNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry storage proof: %w", err)
	}

	return &t.UpdateL2ConfigArgs{
		Config:                        *l2Config,
		L1StorageProof:                l1StorageProof,
		RlpEncodedRegistryAccountData: rlpEncodedRegistryData,
		L1RegistryProof:               l1RegistryProof,
	}, nil
}
