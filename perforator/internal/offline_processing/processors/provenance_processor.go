package processors

import (
	"context"
	"fmt"
	"os"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/offline_processing/models"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/provenance"
	"github.com/yandex/perforator/perforator/pkg/storage/binary"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type ProvenanceProcessor struct {
	l xlog.Logger

	reader      provenance.Reader
	storage     *binary.BinaryStorage
	parseErrors metrics.Counter
	conflicts   metrics.Counter
	missing     metrics.Counter
	saved       metrics.Counter
}

func NewProvenanceProcessor(l xlog.Logger, reg xmetrics.Registry, reader provenance.Reader, storage *binary.BinaryStorage) *ProvenanceProcessor {
	scoped := reg.WithPrefix("build_provenance")
	return &ProvenanceProcessor{
		l:           l.WithName("ProvenanceProcessor"),
		reader:      reader,
		storage:     storage,
		parseErrors: scoped.Counter("parse_errors"),
		conflicts:   scoped.Counter("conflicts"),
		missing:     scoped.Counter("missing"),
		saved:       scoped.Counter("saved"),
	}
}

func (p *ProvenanceProcessor) ProcessBinary(ctx context.Context, _ models.ProcessingTransaction, binaryID, binaryPath string) error {
	binary, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("open binary %s: %w", binaryID, err)
	}
	defer binary.Close()

	info, err := p.reader.Read(binary)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if info == nil {
			p.parseErrors.Inc()
			p.l.Warn(ctx, "Failed to read binary provenance", log.String("build_id", binaryID), log.Error(err))
			return nil
		}
		p.conflicts.Inc()
		p.l.Warn(ctx, "Ambiguous binary provenance", log.String("build_id", binaryID), log.Error(err))
	}
	if info == nil {
		p.missing.Inc()
		return nil
	}

	if info.CommitID == "" && info.BuildTargetPath == "" {
		p.missing.Inc()
		return nil
	}

	attributes := &perforatorstorage.BinaryAttributes{
		BuildInfo: &perforatorstorage.BinaryBuildInfo{
			CommitID:        info.CommitID,
			BuildTargetPath: info.BuildTargetPath,
		},
	}
	if err := p.storage.MergeAttributes(ctx, binaryID, attributes); err != nil {
		return fmt.Errorf("store binary provenance: %w", err)
	}
	p.saved.Inc()
	return nil
}
