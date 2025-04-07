package fallback_prover

import (
	"fmt"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "FALLBACK_PROVER"

const DefaultRegistryAddress = "0x0000000000000000000000000000000000000000"

func prefixEnvVars(names ...string) []string {
	envs := make([]string, 0, len(names))
	for _, name := range names {
		envs = append(envs, EnvVarPrefix+"_"+name)
	}
	return envs
}

var (
	L1HTTPPath = &cli.StringFlag{
		Name:    "l1-http-path",
		Usage:   "HTTP path for an L1 node",
		EnvVars: prefixEnvVars("L1_HTTP_PATH"),
	}
	DstL2HTTPPath = &cli.StringFlag{
		Name:    "dst-l2-http-path",
		Usage:   "HTTP path for a destination L2 eth json rpc endpoint; this is the L2 we are proving state on",
		EnvVars: prefixEnvVars("DST_L2_HTTP_PATH"),
	}
	SrcL2HTTPPath = &cli.StringFlag{
		Name:    "src-l2-http-path",
		Usage:   "HTTP path for a source L2 eth json rpc endpoint; this is the L2 we are proving state of",
		EnvVars: prefixEnvVars("SRC_L1_HTTP_PATH"),
	}
	DstL2ChainID = &cli.Uint64Flag{
		Name:    "dst-l2-chain-id",
		Usage:   "Chain ID for the L2 we are proving state on",
		EnvVars: prefixEnvVars("DST_L2_CHAIN_ID"),
	}
	SrcL2ChainID = &cli.Uint64Flag{
		Name:    "src-l2-chain-id",
		Usage:   "Chain ID for the L2 we are proving state of",
		EnvVars: prefixEnvVars("SRC_L2_CHAIN_ID"),
	}
	SrcL2ContractAddress = &cli.StringFlag{
		Name:    "src-l2-contract-address",
		Usage:   "Contract address we are proving state of, on the source L2",
		EnvVars: prefixEnvVars("SRC_L2_CONTRACT_ADDRESS"),
	}
	SrcL2StorageSlot = &cli.StringFlag{
		Name:    "src-l2-storage-slot",
		Usage:   "Storage slot we are proving state of, on the source L2",
		EnvVars: prefixEnvVars("SRC_L2_STORAGE_SLOT"),
	}
	L1RegistryAddress = &cli.StringFlag{
		Name:    "l1-registry-address",
		Usage:   "Address for the L1 registry; overrides the default",
		EnvVars: prefixEnvVars("SRC_L2_STORAGE_SLOT"),
		Value:   DefaultRegistryAddress,
	}
)

var requiredFlags = []cli.Flag{
	L1HTTPPath,
	SrcL2ChainID,
	SrcL2HTTPPath,
	DstL2ChainID,
	DstL2HTTPPath,
	SrcL2ContractAddress,
	SrcL2StorageSlot,
}

var optionalFlags = []cli.Flag{
	L1RegistryAddress,
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func init() {
	Flags = append(requiredFlags, optionalFlags...)
}

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return opflags.CheckRequiredXor(ctx)
}
