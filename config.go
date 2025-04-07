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

// Config contains the configuration for the native-prover
type Config struct {
	// Additional fields for the prove command
	SrcL2ChainID    uint64
	DstL2ChainID    uint64
	L1HTTPPath      string
	SrcL2RPC        string
	DstL2RPC        string
	SrcAddress      common.Address
	SrcStorageSlot  common.Hash
	RegistryAddress common.Address
}

// NewConfigFromCLI creates a config from the provided *cli.Context
func NewConfigFromCLI(ctx *cli.Context) *Config {
	return &Config{
		L1HTTPPath:      ctx.String(L1HTTPPath.Name),
		SrcL2RPC:        ctx.String(SrcL2HTTPPath.Name),
		DstL2RPC:        ctx.String(DstL2HTTPPath.Name),
		SrcL2ChainID:    ctx.Uint64(SrcL2ChainID.Name),
		DstL2ChainID:    ctx.Uint64(DstL2ChainID.Name),
		SrcAddress:      common.HexToAddress(ctx.String(SrcL2ContractAddress.Name)),
		SrcStorageSlot:  common.HexToHash(ctx.String(SrcL2StorageSlot.Name)),
		RegistryAddress: common.HexToAddress(ctx.String(L1RegistryAddress.Name)),
	}
}
