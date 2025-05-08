package fallback_prover

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/polymerdao/fallback_prover/provers"
	"github.com/polymerdao/fallback_prover/types"
)

// Prover is the main entry point for generating proofs
type Prover struct {
	l1OriginProver     provers.IL1OriginProver
	nativeProver       provers.INativeProver
	l2StorageProver    provers.IStorageProver
	settledStateProver provers.ISettledStateProver
	l2Config           *types.L2ConfigInfo
	l1BlockHashOracle  common.Address
	srcChainID         *big.Int
	configProof        *types.UpdateL2ConfigArgs
	gameIndex          *big.Int
	rootAddress        common.Address
}

// NewProver initializes a new prover with the given RPC endpoints
func NewProver(ctx context.Context, conf *ProveConfig) (*Prover, error) {
	// Set up L1 clients
	l1RPC, err := rpc.Dial(conf.L1HTTPPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	l1Client := ethclient.NewClient(l1RPC)

	// Set up source L2 clients
	srcL2RPC, err := rpc.Dial(conf.SrcL2RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source L2 RPC: %w", err)
	}

	// Set up destination L2 clients
	dstL2RPC, err := rpc.Dial(conf.DstL2RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to destination L2 RPC: %w", err)
	}
	dstL2Client := ethclient.NewClient(dstL2RPC)

	registryProver := provers.NewRegistryProver(l1Client, l1RPC, conf.RegistryAddress)
	l1BlockHashOracle, err := registryProver.GetL1BlockHashOracle(ctx, conf.DstL2ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 block hash oracle: %w", err)
	}

	nativeProver, err := provers.NewNativeProver()
	if err != nil {
		return nil, err
	}

	l2Config, err := registryProver.GetL2Configuration(context.Background(), conf.SrcL2ChainID)
	if err != nil {
		return nil, err
	}
	l2ConfigProof, err := registryProver.GenerateUpdateL2ConfigArgs(ctx, conf.SrcL2ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate L2 config proof: %w", err)
	}

	var settledStateProver provers.ISettledStateProver
	if l2Config.ConfigType == "OPStackBedrock" {
		settledStateProver, err = provers.NewOPStackBedrockProver(l1Client, l1RPC, srcL2RPC)
		if err != nil {
			return nil, err
		}
	} else if l2Config.ConfigType == "OPStackCannon" {
		settledStateProver, err = provers.NewOPStackCannonProver(l1Client, l1RPC, srcL2RPC)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported L2 config type: %s", l2Config.ConfigType)
	}

	index, address, err := settledStateProver.FindLatestResolved(ctx, l2Config)
	if err != nil {
		return nil, fmt.Errorf("failed to find latest resolved info: %w", err)
	}

	return &Prover{
		l1OriginProver:     provers.NewL1OriginProver(l1Client, dstL2Client),
		l2StorageProver:    provers.NewStorageProver(ethclient.NewClient(srcL2RPC), srcL2RPC),
		nativeProver:       nativeProver,
		settledStateProver: settledStateProver,
		l2Config:           l2Config,
		l1BlockHashOracle:  l1BlockHashOracle,
		srcChainID:         big.NewInt(int64(conf.SrcL2ChainID)),
		configProof:        l2ConfigProof,
		gameIndex:          index,
		rootAddress:        address,
	}, nil
}

// GenerateProveCalldata generates the calldata for the NativeProver.prove() function
func (p *Prover) GenerateProveCalldata(
	ctx context.Context,
	params *ProveParams,
) (string, error) {
	rlpEncodedL1Header, l1Header, err := p.GetL1Origin(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	settledStateProof, l2Header, err := p.settledStateProver.GenerateSettledStateProof(
		ctx,
		l1Header.Number,
		p.gameIndex,
		p.rootAddress,
		p.l2Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate %s settled state proof: %w", p.l2Config.ConfigType, err)
	}

	result, err := p.l2StorageProver.GetStorageAt(ctx, params.Address, params.StorageSlot, l2Header.Number)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}
	storageValue := common.HexToHash(result)

	l2StorageProof, rlpEncodedContractAccount, l2AccountProof, err := p.l2StorageProver.GenerateStorageProof(
		ctx,
		params.Address,
		params.StorageSlot,
		l2Header.Root,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	// Create ProveScalarArgs for the Prove call
	proveArgs := types.ProveScalarArgs{
		ChainID:          p.srcChainID,
		ContractAddr:     params.Address,
		StorageSlot:      params.StorageSlot,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2Header.Root,
	}

	rlpEncodedL2Header, err := rlp.EncodeToBytes(l2Header)
	if err != nil {
		return "", fmt.Errorf("failed to encode L2 header: %w", err)
	}

	calldata, err := p.nativeProver.EncodeProveCalldata(
		proveArgs,
		rlpEncodedL1Header,
		rlpEncodedL2Header,
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
	params *ProveParams,
) (string, error) {
	rlpEncodedL1Header, l1Header, err := p.GetL1Origin(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	settledStateProof, l2Header, err := p.settledStateProver.GenerateSettledStateProof(
		ctx,
		l1Header.Number,
		p.gameIndex,
		p.rootAddress,
		p.l2Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate %s settled state proof: %w", p.l2Config.ConfigType, err)
	}

	result, err := p.l2StorageProver.GetStorageAt(ctx, params.Address, params.StorageSlot, l2Header.Number)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}
	storageValue := common.HexToHash(result)

	l2StorageProof, rlpEncodedContractAccount, l2AccountProof, err := p.l2StorageProver.GenerateStorageProof(
		ctx,
		params.Address,
		params.StorageSlot,
		l2Header.Root,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	// Create ProveScalarArgs for the updateAndProve call
	proveArgs := types.ProveScalarArgs{
		ChainID:          p.srcChainID,
		ContractAddr:     params.Address,
		StorageSlot:      params.StorageSlot,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2Header.Root,
	}

	rlpEncodedL2Header, err := rlp.EncodeToBytes(l2Header)
	if err != nil {
		return "", fmt.Errorf("failed to encode L2 header: %w", err)
	}

	calldata, err := p.nativeProver.EncodeUpdateAndProveCalldata(
		*p.configProof,
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
	params *ProveParams,
) (string, error) {
	rlpEncodedL1Header, l1Header, err := p.GetL1Origin(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	settledStateProof, l2Header, err := p.settledStateProver.GenerateSettledStateProof(
		ctx,
		l1Header.Number,
		p.gameIndex,
		p.rootAddress,
		p.l2Config)
	if err != nil {
		return "", fmt.Errorf("failed to generate %s settled state proof: %w", p.l2Config.ConfigType, err)
	}

	result, err := p.l2StorageProver.GetStorageAt(ctx, params.Address, params.StorageSlot, l2Header.Number)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}
	storageValue := common.HexToHash(result)

	l2StorageProof, rlpEncodedContractAccount, l2AccountProof, err := p.l2StorageProver.GenerateStorageProof(
		ctx,
		params.Address,
		params.StorageSlot,
		l2Header.Root,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	// Create ProveScalarArgs for the configureAndProve call
	proveArgs := types.ProveScalarArgs{
		ChainID:          p.srcChainID,
		ContractAddr:     params.Address,
		StorageSlot:      params.StorageSlot,
		StorageValue:     storageValue,
		L2WorldStateRoot: l2Header.Root,
	}

	rlpEncodedL2Header, err := rlp.EncodeToBytes(l2Header)
	if err != nil {
		return "", fmt.Errorf("failed to encode L2 header: %w", err)
	}

	calldata, err := p.nativeProver.EncodeConfigureAndProveCalldata(
		*p.configProof,
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

func (p *Prover) GetL1Origin(ctx context.Context, params *ProveParams) ([]byte, *types2.Header, error) {
	if params.WaitForNewEpoch {
		// Block until we see the L1 origin change
		l1OriginHash, err := p.l1OriginProver.GetL1OriginHash(ctx, p.l1BlockHashOracle)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get initial L1 origin hash: %w", err)
		}
		i := uint(0)

		t := time.NewTicker(time.Duration(params.EpochPollingFreq) * time.Second)
		for {
			select {
			case <-ctx.Done():
				return nil, nil, fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-t.C:
				i++
				newL1OriginHash, err := p.l1OriginProver.GetL1OriginHash(ctx, p.l1BlockHashOracle)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get L1 origin: %w", err)
				}
				if !bytes.Equal(l1OriginHash.Bytes(), newL1OriginHash.Bytes()) {
					return p.l1OriginProver.GetL1Origin(ctx, newL1OriginHash)
				}
				if i > params.EpochPollingTries {
					return nil, nil, fmt.Errorf("timed out waiting for new epoch")
				}
			}
		}
	}
	l1OriginHash, err := p.l1OriginProver.GetL1OriginHash(ctx, p.l1BlockHashOracle)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get L1 origin hash: %w", err)
	}
	return p.l1OriginProver.GetL1Origin(ctx, l1OriginHash)
}
