package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	pprof "github.com/google/pprof/profile"
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type localBatchJob struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type localBatchResult struct {
	Input           string  `json:"input"`
	Output          string  `json:"output"`
	DurationSeconds float64 `json:"duration_seconds"`
	Error           string  `json:"error,omitempty"`
}

// Read and release one profile at a time; the caller keeps the symbolizer and
// its bounded LLVM binary cache alive across all jobs. Never merge profiles.
func runLocalBatch(ctx context.Context, jobs []localBatchJob, report io.Writer, symbolizeProfile func(*pprof.Profile) error) error {
	if err := validateLocalBatch(jobs, ""); err != nil {
		return err
	}

	encoder := json.NewEncoder(report)
	var failures []error
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		started := time.Now()
		err := symbolizeBatchFile(job, symbolizeProfile)
		result := localBatchResult{Input: job.Input, Output: job.Output, DurationSeconds: time.Since(started).Seconds()}
		if err != nil {
			result.Error = err.Error()
			failures = append(failures, fmt.Errorf("%s -> %s: %w", job.Input, job.Output, err))
		}
		// JSONL is written after every job, including failures, so progress and
		// completed outputs survive a subsequent process failure or timeout.
		if err := encoder.Encode(result); err != nil {
			return errors.Join(append(failures, fmt.Errorf("write batch report: %w", err))...)
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("local batch: %d of %d profiles failed: %w", len(failures), len(jobs), errors.Join(failures...))
	}
	return nil
}

func symbolizeBatchFile(job localBatchJob, symbolizeProfile func(*pprof.Profile) error) error {
	input, err := os.Open(job.Input)
	if err != nil {
		return err
	}
	profile, err := pprof.Parse(input)
	closeErr := input.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := symbolizeProfile(profile); err != nil {
		return err
	}

	output, err := os.CreateTemp(filepath.Dir(job.Output), ".symbolized-*")
	if err != nil {
		return err
	}
	defer os.Remove(output.Name())
	err = profile.WriteUncompressed(output)
	closeErr = output.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(output.Name(), job.Output)
}

func validateLocalArgs(cmd *cobra.Command, args []string, options symbolizeLocalArgs) error {
	if len(options.ProfilePaths) == 0 {
		if options.OutputDir != "" || options.ReportPath != "" {
			return fmt.Errorf("--output-dir and --report require --profiles")
		}
		return cobra.ExactArgs(1)(cmd, args)
	}
	if err := cobra.NoArgs(cmd, args); err != nil {
		return err
	}
	if options.OutputDir == "" {
		return fmt.Errorf("--profiles requires --output-dir")
	}
	if cmd.Flags().Changed("output") {
		return fmt.Errorf("--output cannot be combined with --profiles; use --output-dir")
	}
	return nil
}

func symbolizeLocalBatch(ctx context.Context, options symbolizeLocalArgs) error {
	jobs := make([]localBatchJob, 0, len(options.ProfilePaths))
	for _, input := range options.ProfilePaths {
		jobs = append(jobs, localBatchJob{Input: input, Output: filepath.Join(options.OutputDir, filepath.Base(input))})
	}
	if err := validateLocalBatch(jobs, options.ReportPath); err != nil {
		return err
	}
	logger, err := xlog.ForCLI(xlog.CLIConfig{Level: log.InfoLevel})
	if err != nil {
		return err
	}
	started := time.Now()
	provider, err := symbolize.NewFixedBinariesPathProvider(options.LocalBinaryPaths)
	if err != nil {
		return err
	}
	symbolizer, err := symbolize.NewSymbolizer(logger, &nop.Registry{}, nil, nil, symbolize.SymbolizationModeDWARF)
	if err != nil {
		return err
	}
	defer symbolizer.Destroy()
	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		return err
	}
	report := io.Discard
	if options.ReportPath != "" {
		file, err := os.Create(options.ReportPath)
		if err != nil {
			return err
		}
		defer file.Close()
		report = file
	}
	logger.Info(ctx, "Local symbolization started", log.Int("profiles", len(jobs)), log.Int("binaries", len(options.LocalBinaryPaths)))
	err = runLocalBatch(ctx, jobs, report, func(profile *pprof.Profile) error {
		return symbolizer.SymbolizeLocalProfile(ctx, profile, provider, symbolize.NewNilPathProvider())
	})
	logger.Info(ctx, "Local symbolization finished", log.Duration("elapsed", time.Since(started)), log.Bool("success", err == nil))
	return err
}

// Resolve existing links, including links in the parents of not-yet-created outputs.
// A dangling link cannot be resolved safely and is rejected before any writes.
func resolveLocalBatchPath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Abs(resolved)
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if _, statErr := os.Lstat(path); !os.IsNotExist(statErr) {
		return "", err
	}
	// Split instead of Clean/Dir: collapsing symlink/.. before resolving the
	// symlink could change which directory an output refers to.
	parent, name := filepath.Split(strings.TrimRight(path, string(filepath.Separator)))
	if parent == "" {
		parent = "."
	}
	resolved, err = resolveLocalBatchPath(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, name), nil
}

type localBatchPath struct {
	path string
	info os.FileInfo
}

func newLocalBatchPath(path string) (localBatchPath, error) {
	resolved, err := resolveLocalBatchPath(path)
	if err != nil {
		return localBatchPath{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil && !os.IsNotExist(err) {
		return localBatchPath{}, err
	}
	return localBatchPath{path: resolved, info: info}, nil
}

func (p localBatchPath) overlaps(other localBatchPath) bool {
	return p.path == other.path || (p.info != nil && other.info != nil && os.SameFile(p.info, other.info))
}

func validateLocalBatch(jobs []localBatchJob, reportPath string) error {
	// Validate the whole batch and report before writing anything. Compare both
	// resolved names (also for new outputs) and identities of existing files.
	paths := make([]localBatchPath, 0, 2*len(jobs))
	for _, job := range jobs {
		if job.Input == "" || job.Output == "" {
			return fmt.Errorf("batch input and output paths must not be empty")
		}
		path, err := newLocalBatchPath(job.Input)
		if err != nil {
			return err
		}
		paths = append(paths, path)
	}
	for _, job := range jobs {
		path, err := newLocalBatchPath(job.Output)
		if err != nil {
			return err
		}
		for _, other := range paths {
			if path.overlaps(other) {
				return fmt.Errorf("batch output overlaps an input or another output: %s", job.Output)
			}
		}
		paths = append(paths, path)
	}
	if reportPath != "" {
		report, err := newLocalBatchPath(reportPath)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if report.overlaps(path) {
				return fmt.Errorf("report overlaps a profile: %s", reportPath)
			}
		}
	}
	return nil
}
