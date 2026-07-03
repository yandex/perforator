package server

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/yandex/perforator/library/go/core/log"
	binarystorage "github.com/yandex/perforator/perforator/pkg/storage/binary"
)

// debuginfod webapi: https://www.mankier.com/8/debuginfod.
// Both /buildid/HEX/debuginfo and /buildid/HEX/executable currently serve
// the unstripped binary from BinaryStorage; when dedicated split-DWARF
// storage lands, /debuginfo will prefer that bucket.

func (s *PerforatorServer) handleDebuginfod(w http.ResponseWriter, r *http.Request) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.debuginfodRequests.fails.Inc()
		} else {
			s.metrics.debuginfodRequests.successes.Inc()
		}
	}()

	// Storage keys for GNU build-ids are lowercase (hex.EncodeToString output);
	// the debuginfod spec lets clients send either case, so normalize down.
	buildID := strings.ToLower(chi.URLParam(r, "buildid"))
	if buildID == "" {
		err = errors.New("empty build-id")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	logger := s.l.With(log.String("build_id", buildID))

	handle, err := s.binaryDownloader.Acquire(ctx, buildID)
	if err != nil {
		switch {
		case errors.Is(err, binarystorage.ErrNotFound):
			http.Error(w, "build-id not found", http.StatusNotFound)
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			http.Error(w, "request canceled", http.StatusRequestTimeout)
		default:
			logger.Error(ctx, "Failed to acquire binary for debuginfod", log.Error(err))
			http.Error(w, "failed to fetch binary", http.StatusInternalServerError)
		}
		return
	}
	defer handle.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, handle.Path())
}
