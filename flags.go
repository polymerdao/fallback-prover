package fallback_prover

import (
	"fmt"

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
	SrcContractAddress = &cli.StringFlag{
		Name:    "src-contract-address",
		Usage:   "Contract address we are proving state of, on the source L2 or L1",
		EnvVars: prefixEnvVars("SRC_L2_CONTRACT_ADDRESS"),
	}
	SrcStorageSlot = &cli.StringFlag{
		Name:    "src-storage-slot",
		Usage:   "Storage slot we are proving state of, on the source L2 or L1",
		EnvVars: prefixEnvVars("SRC_L2_STORAGE_SLOT"),
	}
	L1RegistryAddress = &cli.StringFlag{
		Name:    "l1-registry-address",
		Usage:   "Address for the L1 registry; overrides the default",
		EnvVars: prefixEnvVars("SRC_L2_STORAGE_SLOT"),
		Value:   DefaultRegistryAddress,
	}
	WaitForNewEpoch = &cli.BoolFlag{
		Name: "wait-for-new-epoch",
		Usage: "Wait for a new L2 epoch before constructing proof if true." +
			"Useful to avoid race condition with L1 blockhash oracle changing",
		EnvVars: prefixEnvVars("WAIT_FOR_NEW_EPOCH"),
		// TODO: update this back to true
		Value: false,
	}
	EpochPollingFreq = &cli.UintFlag{
		Name:    "epoch-polling-freq",
		Usage:   "When wait-for-new-epoch is enabled this configures how often (in seconds) to check for a new epoch",
		EnvVars: prefixEnvVars("EPOCH_POLLING_FREQ"),
		Value:   1,
	}
	EpochPollingTries = &cli.UintFlag{
		Name:    "epoch-polling-tries",
		Usage:   "When wait-for-new-epoch is enabled this configures how many times to query for a new epoch before giving up",
		EnvVars: prefixEnvVars("EPOCH_POLLING_TRIES"),
		Value:   10,
	}
)

var requiredProveFlags = []cli.Flag{
	L1HTTPPath,
	SrcL2ChainID,
	SrcL2HTTPPath,
	DstL2ChainID,
	DstL2HTTPPath,
	SrcContractAddress,
	SrcStorageSlot,
}

var requiredProveL1Flags = []cli.Flag{
	L1HTTPPath,
	DstL2ChainID,
	DstL2HTTPPath,
	SrcContractAddress,
	SrcStorageSlot,
}

var optionalFlags = []cli.Flag{
	L1RegistryAddress,
	WaitForNewEpoch,
	EpochPollingFreq,
	EpochPollingTries,
}

// L2Flags contains the list of configuration options available for the prove commands
var L2Flags []cli.Flag

// L1Flags contains the list of configuration options available for the proveL1 commands
var L1Flags []cli.Flag

func init() {
	L2Flags = append(requiredProveFlags, optionalFlags...)
	L1Flags = append(requiredProveL1Flags, optionalFlags...)
}

func CheckRequiredL2(ctx *cli.Context) error {
	for _, f := range requiredProveFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}

func CheckRequiredL1(ctx *cli.Context) error {
	for _, f := range requiredProveL1Flags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
