package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:          "cpu_burner --duration DURATION",
		Short:        "Burn some CPU for taking samples",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run()
		},
	}

	duration time.Duration
)

func init() {
	rootCmd.Flags().DurationVar(&duration, "duration", time.Second*10, "duration of cpu burning")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	//nolint:sa5004
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
}
