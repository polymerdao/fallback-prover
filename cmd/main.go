package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/polymerdao/fallback_prover"
)

var (
	GitCommit = ""
	GitDate   = ""
)

var VersionWithMeta = fallback_prover.FormattedVersion(GitCommit, GitDate)

func main() {
	log.SetDefault(log.NewLogger(log.LogfmtHandlerWithLevel(os.Stderr, log.LevelInfo)))

	app := cli.NewApp()
	app.Version = VersionWithMeta
	app.Flags = []cli.Flag{}
	app.Name = "native-proof"
	app.Usage = "Tool for generating native proofs"
	app.Description = "CLI tool that generates calldata for NativeProver.prove() function"
	app.Commands = []*cli.Command{
		ProveNativeCmd,
		ProveL1NativeCmd,
	}

	// Create a context that gets canceled on interrupt signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		cancel()
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

var ProveNativeCmd = &cli.Command{
	Name:        "proveNative",
	Usage:       "Generate proof calldata for NativeProver.proveNative() function",
	Description: "Generate the calldata for a transaction calling the NativeProver.proveNative() function",
	Action:      proveNative,
	Flags:       fallback_prover.L2Flags,
}

var ProveL1NativeCmd = &cli.Command{
	Name:        "proveNativeL1",
	Usage:       "Generate proof calldata for NativeProver.proveL1Native() function",
	Description: "Generate the calldata for a transaction calling the NativeProver.proveL1Native() function",
	Action:      proveL1Native,
	Flags:       fallback_prover.L1Flags,
}

func proveL1Native(c *cli.Context) error {
	if err := fallback_prover.CheckRequiredL1(c); err != nil {
		return err
	}

	config := fallback_prover.NewL1ConfigFromCLI(c)
	params := fallback_prover.NewParamsFromCLI(c)

	log.Info("Generating proveL1() calldata",
		"dstL2ChainID", config.DstL2ChainID,
		"srcAddress", params.Address,
		"srcStorageSlot", params.StorageSlot)

	// Initialize the prover
	prover, err := fallback_prover.NewL1Prover(
		c.Context,
		config,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize prover: %w", err)
	}

	// Generate proof calldata
	calldata, err := prover.GenerateProveL1Calldata(
		c.Context,
		params,
	)
	if err != nil {
		return fmt.Errorf("failed to generate proof calldata: %w", err)
	}

	// Output the calldata
	fmt.Println(calldata)
	return nil
}

func proveNative(c *cli.Context) error {
	if err := fallback_prover.CheckRequiredL2(c); err != nil {
		return err
	}

	config := fallback_prover.NewConfigFromCLI(c)
	params := fallback_prover.NewParamsFromCLI(c)

	log.Info("Generating proveNative() calldata",
		"srcL2ChainID", config.SrcL2ChainID,
		"dstL2ChainID", config.DstL2ChainID,
		"srcAddress", params.Address,
		"srcStorageSlot", params.StorageSlot)

	// Initialize the prover
	prover, err := fallback_prover.NewProver(
		c.Context,
		config,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize prover: %w", err)
	}

	// Generate proveNative calldata
	calldata, err := prover.GenerateProveNativeCalldata(
		c.Context,
		params,
	)
	if err != nil {
		return fmt.Errorf("failed to generate proveNative calldata: %w", err)
	}

	// Output the calldata
	fmt.Println(calldata)
	return nil
}
