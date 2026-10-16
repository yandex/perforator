package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/pkg/cprofile"
	profilepb "github.com/yandex/perforator/perforator/proto/profile"
)

func makeMergeCmd() *cobra.Command {
	var threads int
	cmd := &cobra.Command{
		Use:   "merge profile.pprof...",
		Short: "Merge local raw/gzip pprof files and write the merged pprof to stdout",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, paths []string) error {
			if threads < 1 {
				return fmt.Errorf("--threads must be positive")
			}
			return mergeLocalProfiles(cmd.Context(), paths, threads, cmd.OutOrStdout())
		},
	}
	cmd.Flags().IntVar(&threads, "threads", 1, "Merge threads; inputs are read one at a time")
	return cmd
}

func mergeLocalProfiles(ctx context.Context, paths []string, threads int, output io.Writer) error {
	manager, err := cprofile.NewMergeManager(threads)
	if err != nil {
		return fmt.Errorf("initialize merge: %w", err)
	}
	session, err := manager.Start(&profilepb.MergeOptions{})
	if err != nil {
		return err
	}
	defer session.Close()
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read profile %q: %w", path, err)
		}
		if err := session.AddPProfProfile(data); err != nil {
			return fmt.Errorf("merge profile %q: %w", path, err)
		}
	}
	merged, err := session.Finish()
	if err != nil {
		return fmt.Errorf("finish merge: %w", err)
	}
	defer merged.Free()
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := merged.MarshalPProf()
	if err != nil {
		return fmt.Errorf("serialize merged profile: %w", err)
	}
	_, err = output.Write(data)
	return err
}
