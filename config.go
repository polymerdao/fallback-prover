package fallback_prover

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// L2ConfigInfo contains the configuration for an L2 chain
type L2ConfigInfo struct {
	ConfigType   string
	Addresses    []common.Address
	StorageSlots []uint64
}

// ProveConfig contains the configuration for proving a storage slot on a src L2
type ProveConfig struct {
	SrcL2ChainID    uint64
	DstL2ChainID    uint64
	L1HTTPPath      string
	SrcL2RPC        string
	DstL2RPC        string
	RegistryAddress common.Address
	WaitForNewEpoch bool
}

// ProveL1Config contains the configuration for proving a storage slot on an L1
type ProveL1Config struct {
	DstL2ChainID    uint64
	L1HTTPPath      string
	DstL2RPC        string
	RegistryAddress common.Address
}

type ProveParams struct {
	Address           common.Address
	StorageSlot       common.Hash
	WaitForNewEpoch   bool
	EpochPollingFreq  uint
	EpochPollingTries uint
}

// NewConfigFromCLI creates a config from the provided *cli.Context
func NewConfigFromCLI(ctx *cli.Context) *ProveConfig {
	return &ProveConfig{
		L1HTTPPath:      ctx.String(L1HTTPPath.Name),
		SrcL2RPC:        ctx.String(SrcL2HTTPPath.Name),
		DstL2RPC:        ctx.String(DstL2HTTPPath.Name),
		SrcL2ChainID:    ctx.Uint64(SrcL2ChainID.Name),
		DstL2ChainID:    ctx.Uint64(DstL2ChainID.Name),
		RegistryAddress: common.HexToAddress(ctx.String(L1RegistryAddress.Name)),
		WaitForNewEpoch: ctx.Bool(WaitForNewEpoch.Name),
	}
}

func NewL1ConfigFromCLI(ctx *cli.Context) *ProveL1Config {
	return &ProveL1Config{
		L1HTTPPath:      ctx.String(L1HTTPPath.Name),
		DstL2RPC:        ctx.String(DstL2HTTPPath.Name),
		DstL2ChainID:    ctx.Uint64(DstL2ChainID.Name),
		RegistryAddress: common.HexToAddress(ctx.String(L1RegistryAddress.Name)),
	}
}

func NewParamsFromCLI(ctx *cli.Context) *ProveParams {
	return &ProveParams{
		Address:           common.HexToAddress(ctx.String(SrcContractAddress.Name)),
		StorageSlot:       common.HexToHash(ctx.String(SrcStorageSlot.Name)),
		WaitForNewEpoch:   ctx.Bool(WaitForNewEpoch.Name),
		EpochPollingFreq:  ctx.Uint(EpochPollingFreq.Name),
		EpochPollingTries: ctx.Uint(EpochPollingTries.Name),
	}
}
