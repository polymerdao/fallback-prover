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
		ProveCmd,
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

var ProveCmd = &cli.Command{
	Name:        "prove",
	Usage:       "Generate proof calldata for NativeProver.prove() function",
	Description: "Generate the calldata for a transaction calling the NativeProver.prove() function",
	Action:      prove,
	Flags:       fallback_prover.Flags,
}

func prove(c *cli.Context) error {
	if err := fallback_prover.CheckRequired(c); err != nil {
		return err
	}

	config := fallback_prover.NewConfigFromCLI(c)

	log.Info("Generating proof calldata",
		"srcL2ChainID", config.SrcL2ChainID,
		"dstL2ChainID", config.DstL2ChainID,
		"srcAddress", config.SrcAddress,
		"srcStorageSlot", config.SrcStorageSlot)

	// Initialize the prover
	prover, err := fallback_prover.NewProver(
		config.L1HTTPPath,
		config.SrcL2RPC,
		config.DstL2RPC,
		config.SrcL2ChainID,
		config.RegistryAddress.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize prover: %w", err)
	}

	// Generate proof calldata
	calldata, err := prover.GenerateProofCalldata(
		c.Context,
		config.SrcL2ChainID,
		config.DstL2ChainID,
		config.SrcAddress.String(),
		config.SrcStorageSlot.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to generate proof calldata: %w", err)
	}

	// Output the calldata
	fmt.Println(calldata)
	return nil
}
