package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/resource"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	service "github.com/yandex/perforator/perforator/internal/web"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	rootCmd = &cobra.Command{
		Use:           "web",
		Short:         "Starts a web service to serve ui",
		Long:          "This service is able to share static ui files, make requests to perforator proxy /api route, fetch render profiles from s3 bucket",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run()
		},
	}

	configPath string
	logLevel   string
)

var (
	storageConfigForValidationPath string

	storageValidateConfigCmd = &cobra.Command{
		Use:   "validate-config",
		Short: "Validate storage config",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := service.ParseConfig(storageConfigForValidationPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%#v\n", config)
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to web service config")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level (must be one of `debug`, `info`, `warn`, `error`)")

	cobrabuildinfo.Init(rootCmd)

	rootCmd.MarkFlagsOneRequired("config")
	rootCmd.AddCommand(storageValidateConfigCmd)

	storageValidateConfigCmd.Flags().StringVarP(&storageConfigForValidationPath, "config", "c", "", "path to web service config")
	storageValidateConfigCmd.MarkFlagsOneRequired("config")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	logger, err := setupLogger(logLevel)
	if err != nil {
		return err
	}

	cfg, err := service.ParseConfig(configPath)
	if err != nil {
		return err
	}

	reg := xmetrics.NewRegistry()

	fs := afero.NewMemMapFs()

	uiTar := resource.Get("ui.tar")

	if err := untarToFs(ctx, logger, fs, uiTar); err != nil {
		return err
	}

	serv, err := service.NewWebService(cfg, logger, reg, fs)
	if err != nil {
		return err
	}

	return serv.Run(
		ctx,
		cfg.PortsConfig,
	)
}

func setupLogger(logLevel string) (xlog.Logger, error) {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}

	logger, err := xlog.TryNew(zap.NewDeployLogger(level))
	if err != nil {
		return nil, fmt.Errorf("can't create logger: %w", err)
	}

	return logger, nil
}

// untarToFs untars a tar archive from data into the provided Afero filesystem
func untarToFs(ctx context.Context, logger xlog.Logger, fs afero.Fs, data []byte) error {
	tr := tar.NewReader(bytes.NewReader(data))

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar archive: %w", err)
		}

		path := filepath.Clean(hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("creating directory %s: %w", path, err)
			}
		case tar.TypeReg:
			dir := filepath.Dir(path)
			if err := fs.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating parent directories for %s: %w", path, err)
			}

			file, err := fs.OpenFile(path, os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("creating file %s: %w", path, err)
			}
			defer file.Close()
			if _, err := io.Copy(file, tr); err != nil {
				return fmt.Errorf("writing to file %s: %w", path, err)
			}
		default:
			logger.Warn(ctx, "Skipping unsupported file type in archive", log.String("header", hdr.Name))
		}
	}
	err := afero.Walk(fs, "dist", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.HasSuffix(info.Name(), ".map") {
			return nil
		}
		logger.Info(ctx, "In-memory FS contains:", log.String("path", path))
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk ui fs: %w", err)
	}
	return nil
}
