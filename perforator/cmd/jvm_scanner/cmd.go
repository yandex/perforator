package main

import (
	"context"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmscanner"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmsupportservice"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var mapPrefix string
var socketPath string
var debugLogs bool

func makeSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			var level log.Level
			if debugLogs {
				level = log.DebugLevel
			} else {
				level = log.InfoLevel
			}
			logger, err := xlog.ForCLI(xlog.CLIConfig{
				Level:  level,
				Format: xlog.LogFormatJson,
			})
			if err != nil {
				return fmt.Errorf("failed to create logger: %w", err)
			}

			return run(cmd.Context(), logger)
		},
	}
	cmd.Flags().StringVar(&mapPrefix, "map-prefix", "", "Prefix for all pinned maps, i.e. effective-path is ${map-prefix}${map-name}")
	must.Must(cmd.MarkFlagRequired("map-prefix"))

	cmd.Flags().StringVar(&socketPath, "socket-path", "", "Path to UDS to serve on")
	must.Must(cmd.MarkFlagRequired("socket-path"))

	cmd.Flags().BoolVar(&debugLogs, "debug-logging", false, "Enable debug logs")

	return cmd
}

func run(ctx context.Context, logger xlog.Logger) error {
	jvmProcesses, err := ebpf.LoadPinnedMap(mapPrefix+"jvm_processes", &ebpf.LoadPinOptions{})
	if err != nil {
		return fmt.Errorf("failed to load jvm_processes map: %w", err)
	}
	processInfo, err := ebpf.LoadPinnedMap(mapPrefix+"process_info", &ebpf.LoadPinOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to load process_info map: %w", err)
	}

	bpf := programstate.New(&unwinder.Maps{
		JvmProcesses: jvmProcesses,
		ProcessInfo:  processInfo,
	}, nil)

	scanner := jvmscanner.New(logger.WithName("scanner"), bpf)
	s := jvmsupportservice.New(
		logger.WithName("api"),
		scanner,
		bpf,
		jvmsupportservice.Options{
			SocketPath: socketPath,
		},
	)
	return s.Run(ctx)
}
