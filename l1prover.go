package fallback_prover

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/polymerdao/fallback_prover/provers"
	"github.com/polymerdao/fallback_prover/types"
)

// L1Prover is the main entry point for generating L1 proofs
type L1Prover struct {
	l1OriginProver    provers.IL1OriginProver
	nativeProver      provers.INativeProver
	l1StorageProver   provers.IStorageProver
	l1BlockHashOracle common.Address
}

// NewL1Prover initializes a new prover with the given RPC endpoints
func NewL1Prover(ctx context.Context, conf *ProveL1Config) (*L1Prover, error) {
	// Set up L1 clients
	l1RPC, err := rpc.Dial(conf.L1HTTPPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	l1Client := ethclient.NewClient(l1RPC)

	// Set up destination L2 clients
	dstL2RPC, err := rpc.Dial(conf.DstL2RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to destination L2 RPC: %w", err)
	}
	dstL2Client := ethclient.NewClient(dstL2RPC)

	registryProver := provers.NewRegistryProver(l1Client, l1RPC, conf.RegistryAddress)
	nativeProver, err := provers.NewNativeProver()
	if err != nil {
		return nil, fmt.Errorf("failed to create native prover: %w", err)
	}

	l1BlockHashOracle, err := registryProver.GetL1BlockHashOracle(ctx, conf.DstL2ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get L1 block hash oracle: %w", err)
	}

	return &L1Prover{
		l1OriginProver:    provers.NewL1OriginProver(l1Client, dstL2Client),
		l1StorageProver:   provers.NewStorageProver(l1Client, l1RPC),
		nativeProver:      nativeProver,
		l1BlockHashOracle: l1BlockHashOracle,
	}, nil
}

// GenerateProveL1Calldata generates the calldata for the NativeProver.proveL1() function
func (p *L1Prover) GenerateProveL1Calldata(
	ctx context.Context,
	params *ProveParams,
) (string, error) {
	rlpEncodedL1Header, l1Header, err := p.GetL1Origin(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get L1 origin: %w", err)
	}

	result, err := p.l1StorageProver.GetStorageAt(ctx, params.Address, params.StorageSlot, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get storage value: %w", err)
	}

	storageValue := common.HexToHash(result)
	l1StorageProof, rlpEncodedContractAccount, l1AccountProof, err := p.l1StorageProver.GenerateStorageProof(
		ctx,
		params.Address,
		params.StorageSlot,
		l1Header.Number,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate storage proof: %w", err)
	}

	proveArgs := types.ProveL1ScalarArgs{
		ContractAddr:     params.Address,
		StorageSlot:      params.StorageSlot,
		StorageValue:     storageValue,
		L1WorldStateRoot: l1Header.Root,
	}

	calldata, err := p.nativeProver.EncodeProveL1NativeCalldata(
		proveArgs,
		rlpEncodedL1Header,
		l1StorageProof,
		rlpEncodedContractAccount,
		l1AccountProof,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack calldata: %w", err)
	}

	// Return the calldata as a hex string
	return "0x" + common.Bytes2Hex(calldata), nil
}

func (p *L1Prover) GetL1Origin(ctx context.Context, params *ProveParams) ([]byte, *types2.Header, error) {
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
