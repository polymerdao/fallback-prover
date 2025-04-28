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
		UpdateAndProveCmd,
		ConfigureAndProveCmd,
		ProveL1Cmd,
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
	Flags:       fallback_prover.L2Flags,
}

var UpdateAndProveCmd = &cli.Command{
	Name:        "update-and-prove",
	Usage:       "Generate proof calldata for NativeProver.updateAndProve() function",
	Description: "Generate the calldata for a transaction calling the NativeProver.updateAndProve() function",
	Action:      updateAndProve,
	Flags:       fallback_prover.L2Flags,
}

var ConfigureAndProveCmd = &cli.Command{
	Name:        "configure-and-prove",
	Usage:       "Generate proof calldata for NativeProver.configureAndProve() function",
	Description: "Generate the calldata for a transaction calling the NativeProver.configureAndProve() function",
	Action:      configureAndProve,
	Flags:       fallback_prover.L2Flags,
}

var ProveL1Cmd = &cli.Command{
	Name:        "proveL1",
	Usage:       "Generate proof calldata for NativeProver.proveL1() function",
	Description: "Generate the calldata for a transaction calling the NativeProver.proveL1() function",
	Action:      proveL1,
	Flags:       fallback_prover.L1Flags,
}

func prove(c *cli.Context) error {
	if err := fallback_prover.CheckRequiredL2(c); err != nil {
		return err
	}

	config := fallback_prover.NewConfigFromCLI(c)
	params := fallback_prover.NewParamsFromCLI(c)

	log.Info("Generating prove() calldata",
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

	// Generate proof calldata
	calldata, err := prover.GenerateProveCalldata(
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

func proveL1(c *cli.Context) error {
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

func updateAndProve(c *cli.Context) error {
	if err := fallback_prover.CheckRequiredL2(c); err != nil {
		return err
	}

	config := fallback_prover.NewConfigFromCLI(c)
	params := fallback_prover.NewParamsFromCLI(c)

	log.Info("Generating updateAndProve() calldata",
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

	// Generate updateAndProve calldata
	calldata, err := prover.GenerateUpdateAndProveCalldata(
		c.Context,
		params,
	)
	if err != nil {
		return fmt.Errorf("failed to generate updateAndProve calldata: %w", err)
	}

	// Output the calldata
	fmt.Println(calldata)
	return nil
}

func configureAndProve(c *cli.Context) error {
	if err := fallback_prover.CheckRequiredL2(c); err != nil {
		return err
	}

	config := fallback_prover.NewConfigFromCLI(c)
	params := fallback_prover.NewParamsFromCLI(c)

	log.Info("Generating configureAndProve() calldata",
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

	// Generate configureAndProve calldata
	calldata, err := prover.GenerateConfigureAndProveCalldata(
		c.Context,
		params,
	)
	if err != nil {
		return fmt.Errorf("failed to generate configureAndProve calldata: %w", err)
	}

	// Output the calldata
	fmt.Println(calldata)
	return nil
}
