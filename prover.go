package fallback_prover

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/polymerdao/fallback_prover/provers"
	"github.com/polymerdao/fallback_prover/types"
)

// Prover is the main entry point for generating proofs
type Prover struct {
	registryProver     provers.IRegistryProver
	l1OriginProver     provers.IL1OriginProver
	nativeProver       provers.INativeProver
	l2StorageProver    provers.IStorageProver
	settledStateProver provers.ISettledStateProver
	l2Config           *types.L2ConfigInfo
}

// NewProver initializes a new prover with the given RPC endpoints
func NewProver(l1RPCEndpoint, srcL2RPCEndpoint, dstL2RPCEndpoint string, srcL2ChainID uint64, registryAddr string) (*Prover, error) {
	// Set up L1 clients
	l1RPC, err := rpc.Dial(l1RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	l1Client := ethclient.NewClient(l1RPC)

	// Set up source L2 clients
	srcL2RPC, err := rpc.Dial(srcL2RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source L2 RPC: %w", err)
	}
	srcL2Client := ethclient.NewClient(srcL2RPC)

	// Set up destination L2 clients
	dstL2RPC, err := rpc.Dial(dstL2RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to destination L2 RPC: %w", err)
	}
	dstL2Client := ethclient.NewClient(dstL2RPC)

	registryProver := provers.NewRegistryProver(l1Client, l1RPC, common.HexToAddress(registryAddr))
	nativeProver, err := provers.NewNativeProver()
	if err != nil {
		return nil, err
	}

	l2Config, err := registryProver.GetL2Configuration(context.Background(), srcL2ChainID)
	if err != nil {
		return nil, err
	}

	var settledStateProver provers.ISettledStateProver
	if l2Config.ConfigType == "OPStackBedrock" {
		settledStateProver, err = provers.NewOPStackBedrockProver(l1Client, l1RPC, srcL2Client, srcL2RPC)
		if err != nil {
			return nil, err
		}
	} else if l2Config.ConfigType == "OPStackCannon" {
		settledStateProver, err = provers.NewOPStackCannonProver(l1Client, l1RPC, srcL2Client, srcL2RPC)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported L2 config type: %s", l2Config.ConfigType)
	}
	return &Prover{
		registryProver:     registryProver,
		l1OriginProver:     provers.NewL1OriginProver(l1Client, dstL2Client),
		l2StorageProver:    provers.NewStorageProver(srcL2Client, srcL2RPC),
		nativeProver:       nativeProver,
		settledStateProver: settledStateProver,
		l2Config:           l2Config,
	}, nil
}

// GenerateProofCalldata generates the calldata for the NativeProver.prove() function
func (p *Prover) GenerateProofCalldata(
	ctx context.Context,
	srcL2ChainID uint64,
	dstL2ChainID uint64,
	srcAddress string,
	srcStorageSlot string,
) (string, error) {
	l1BlockHashOracle, err := p.registryProver.GetL1BlockHashOracle(ctx, dstL2ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 block hash oracle: %w", err)
	}

	rlpEncodedL1Header, _, err := p.l1OriginProver.ProveL1Origin(ctx, l1BlockHashOracle)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	contractAddr := common.HexToAddress(srcAddress)
	slotHash := common.HexToHash(srcStorageSlot)

	result, err := p.l2StorageProver.GetStorageAt(ctx, contractAddr, slotHash, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}

	storageValue := common.HexToHash(result)

	var settledStateProof []byte
	var l2WorldStateRoot common.Hash
	var rlpEncodedL2Header []byte

	settledStateProof, l2WorldStateRoot, rlpEncodedL2Header, err = p.settledStateProver.GenerateSettledStateProof(
		ctx,
		p.l2Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate %s settled state proof: %w", p.l2Config.ConfigType, err)
	}

	l2StorageProof, rlpEncodedContractAccount, l2AccountProof, err := p.l2StorageProver.GenerateStorageProof(
		ctx,
		contractAddr,
		slotHash,
		l2WorldStateRoot,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	calldata, err := p.nativeProver.EncodeProveCalldata(
		big.NewInt(int64(srcL2ChainID)),
		contractAddr,
		slotHash,
		storageValue,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
		l2WorldStateRoot,
		settledStateProof,
		l2StorageProof,
		rlpEncodedContractAccount,
		l2AccountProof,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack calldata: %w", err)
	}

	// Return the calldata as a hex string
	return "0x" + common.Bytes2Hex(calldata), nil
}

// GenerateUpdateAndProveCalldata generates the calldata for the NativeProver.updateAndProve() function
func (p *Prover) GenerateUpdateAndProveCalldata(
	ctx context.Context,
	srcL2ChainID uint64,
	dstL2ChainID uint64,
	srcAddress string,
	srcStorageSlot string,
) (string, error) {
	l1BlockHashOracle, err := p.registryProver.GetL1BlockHashOracle(ctx, dstL2ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 block hash oracle: %w", err)
	}

	rlpEncodedL1Header, _, err := p.l1OriginProver.ProveL1Origin(ctx, l1BlockHashOracle)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	// Generate the UpdateL2ConfigArgs from the registry
	updateConfig, err := p.registryProver.GenerateUpdateL2ConfigArgs(ctx, srcL2ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to generate update L2 config args: %w", err)
	}

	contractAddr := common.HexToAddress(srcAddress)
	slotHash := common.HexToHash(srcStorageSlot)

	result, err := p.l2StorageProver.GetStorageAt(ctx, contractAddr, slotHash, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}

	storageValue := common.HexToHash(result)

	var settledStateProof []byte
	var l2WorldStateRoot common.Hash
	var rlpEncodedL2Header []byte

	settledStateProof, l2WorldStateRoot, rlpEncodedL2Header, err = p.settledStateProver.GenerateSettledStateProof(
		ctx,
		p.l2Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate %s settled state proof: %w", p.l2Config.ConfigType, err)
	}

	l2StorageProof, rlpEncodedContractAccount, l2AccountProof, err := p.l2StorageProver.GenerateStorageProof(
		ctx,
		contractAddr,
		slotHash,
		l2WorldStateRoot,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	// Create ProveScalarArgs for the updateAndProve call
	proveArgs := types.ProveScalarArgs{
		ChainID:          big.NewInt(int64(srcL2ChainID)),
		ContractAddr:     contractAddr,
		StorageSlot:      slotHash,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2WorldStateRoot,
	}

	calldata, err := p.nativeProver.EncodeUpdateAndProveCalldata(
		*updateConfig,
		proveArgs,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
		settledStateProof,
		l2StorageProof,
		rlpEncodedContractAccount,
		l2AccountProof,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack updateAndProve calldata: %w", err)
	}

	// Return the calldata as a hex string
	return "0x" + common.Bytes2Hex(calldata), nil
}

// GenerateConfigureAndProveCalldata generates the calldata for the NativeProver.configureAndProve() function
func (p *Prover) GenerateConfigureAndProveCalldata(
	ctx context.Context,
	srcL2ChainID uint64,
	dstL2ChainID uint64,
	srcAddress string,
	srcStorageSlot string,
) (string, error) {
	l1BlockHashOracle, err := p.registryProver.GetL1BlockHashOracle(ctx, dstL2ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 block hash oracle: %w", err)
	}

	rlpEncodedL1Header, _, err := p.l1OriginProver.ProveL1Origin(ctx, l1BlockHashOracle)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	// Generate the UpdateL2ConfigArgs from the registry
	updateConfig, err := p.registryProver.GenerateUpdateL2ConfigArgs(ctx, srcL2ChainID)
	if err != nil {
		return "", fmt.Errorf("failed to generate update L2 config args: %w", err)
	}

	contractAddr := common.HexToAddress(srcAddress)
	slotHash := common.HexToHash(srcStorageSlot)

	result, err := p.l2StorageProver.GetStorageAt(ctx, contractAddr, slotHash, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}

	storageValue := common.HexToHash(result)

	var settledStateProof []byte
	var l2WorldStateRoot common.Hash
	var rlpEncodedL2Header []byte

	settledStateProof, l2WorldStateRoot, rlpEncodedL2Header, err = p.settledStateProver.GenerateSettledStateProof(
		ctx,
		p.l2Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate %s settled state proof: %w", p.l2Config.ConfigType, err)
	}

	l2StorageProof, rlpEncodedContractAccount, l2AccountProof, err := p.l2StorageProver.GenerateStorageProof(
		ctx,
		contractAddr,
		slotHash,
		l2WorldStateRoot,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	// Create ProveScalarArgs for the configureAndProve call
	proveArgs := types.ProveScalarArgs{
		ChainID:          big.NewInt(int64(srcL2ChainID)),
		ContractAddr:     contractAddr,
		StorageSlot:      slotHash,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2WorldStateRoot,
	}

	calldata, err := p.nativeProver.EncodeConfigureAndProveCalldata(
		*updateConfig,
		proveArgs,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
		settledStateProof,
		l2StorageProof,
		rlpEncodedContractAccount,
		l2AccountProof,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack configureAndProve calldata: %w", err)
	}

	// Return the calldata as a hex string
	return "0x" + common.Bytes2Hex(calldata), nil
}
