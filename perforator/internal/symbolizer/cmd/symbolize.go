package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	pprof "github.com/google/pprof/profile"
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type (
	symbolizeLocalArgs struct {
		OutputPath             string
		LocalBinaryStoragePath string
	}

	symbolizeStorageArgs struct {
		OutputPath             string
		ConfigPath             string
		LocalBinaryStoragePath string
	}
)

var (
	configPath  string
	localArgs   symbolizeLocalArgs
	storageArgs symbolizeStorageArgs

	symbolizeCmd = &cobra.Command{
		Use:   "symbolize {local | storage}",
		Short: "Symbolize profile",
	}

	symbolizeLocalCmd = &cobra.Command{
		Use:   "local <profile_path>",
		Short: "Symbolize profile from fs",
		RunE: func(_ *cobra.Command, args []string) error {
			profilePath := args[0]

			logger, err := xlog.TryNew(zap.NewDeployLogger(log.DebugLevel))
			if err != nil {
				return err
			}

			profileFile, err := os.Open(profilePath)
			if err != nil {
				return err
			}
			defer profileFile.Close()

			profile, err := pprof.Parse(profileFile)
			if err != nil {
				return err
			}

			outputFile, err := os.Create(localArgs.OutputPath)
			if err != nil {
				return err
			}
			defer outputFile.Close()

			symbolizer, err := symbolize.NewSymbolizer(
				logger,
				&nop.Registry{},
				nil,
				nil,
				symbolize.SymbolizationModeDWARF,
			)
			if err != nil {
				return err
			}
			defer symbolizer.Destroy()

			err = symbolizer.SymbolizeLocalProfile(context.Background(), profile)
			if err != nil {
				return err
			}

			return profile.WriteUncompressed(outputFile)
		},
	}

	symbolizeStorageCmd = &cobra.Command{
		Use:   "storage [profile_id]",
		Short: "Symbolize profile in storage described by config (usually remote storage)",
		RunE: func(_ *cobra.Command, args []string) error {
			profileID := args[0]

			logger, err := xlog.TryNew(zap.NewDeployLogger(log.DebugLevel))
			if err != nil {
				return err
			}

			outputFile, err := os.Create(storageArgs.OutputPath)
			if err != nil {
				return err
			}
			defer outputFile.Close()

			initCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
			defer cancel()
			bundle, err := bundle.NewStorageBundleFromConfig(initCtx, logger, &nop.Registry{}, configPath)
			if err != nil {
				return err
			}

			// TODO : downloaderInstance.RunBackgroundDownloader?
			_, binaryDownloader, gsymDownloader, err := downloader.CreateDownloaders(
				&asyncfilecache.Config{
					MaxSize:  "100G",
					MaxItems: 10000000,
					RootPath: "./binaries",
				},
				20,
				logger, nop.Registry{},
				bundle.BinaryStorage.Binary(),
				bundle.BinaryStorage.GSYM(),
			)
			if err != nil {
				return err
			}

			symbolizer, err := symbolize.NewSymbolizer(
				logger,
				&nop.Registry{},
				binaryDownloader,
				gsymDownloader,
				symbolize.SymbolizationModeDWARF,
			)
			if err != nil {
				return err
			}
			defer symbolizer.Destroy()

			storageProfile, err := bundle.ProfileStorage.GetProfiles(context.Background(), []meta.ProfileID{profileID}, false)
			if err != nil {
				return err
			}
			if len(storageProfile) == 0 {
				return fmt.Errorf("profile `%s` cannot be found in storage", profileID)
			}

			profile, err := pprof.ParseData(storageProfile[0].Body)
			if err != nil {
				return err
			}

			symbolizedProfile, err := symbolizer.SymbolizeStorageProfile(
				context.Background(),
				profile,
				nil,
			)
			if err != nil {
				return err
			}

			return symbolizedProfile.WriteUncompressed(outputFile)
		},
	}
)

func init() {
	symbolizeStorageCmd.Flags().StringVarP(
		&storageArgs.OutputPath,
		"output",
		"o",
		"symbolized_profile.pprof",
		"Path to uncompressed symbolized output",
	)
	must.Must(symbolizeStorageCmd.MarkFlagFilename("output"))
	symbolizeStorageCmd.Flags().StringVarP(
		&storageArgs.ConfigPath,
		"config",
		"c",
		"",
		"Path to the storage v2 config (binary storage + profile storage)",
	)
	must.Must(symbolizeStorageCmd.MarkFlagFilename("config"))
	symbolizeStorageCmd.Flags().StringVarP(
		&storageArgs.LocalBinaryStoragePath,
		"local-bin",
		"l",
		"./",
		"Path to the local binary cache dir",
	)
	must.Must(symbolizeStorageCmd.MarkFlagDirname("local-bin"))

	symbolizeLocalCmd.Flags().StringVarP(
		&localArgs.OutputPath,
		"output",
		"o",
		"symbolized_profile.pprof",
		"Path to uncompressed symbolized output",
	)
	symbolizeLocalCmd.Flags().StringVarP(
		&localArgs.LocalBinaryStoragePath,
		"local-bin",
		"l",
		"./",
		"Path to the local binary cache dir",
	)
	must.Must(symbolizeLocalCmd.MarkFlagFilename("output"))

	symbolizeCmd.AddCommand(symbolizeLocalCmd)
	symbolizeCmd.AddCommand(symbolizeStorageCmd)

	rootCmd.AddCommand(symbolizeCmd)
}
