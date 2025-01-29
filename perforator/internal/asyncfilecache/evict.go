package asyncfilecache

import (
	"os"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/weightedlru"
)

func evictLRUCallback(l log.Logger) func(key, value interface{}) {
	return func(key, val interface{}) {
		path := key.(string)

		entry := val.(*weightedlru.CacheItem).Value.(*cacheEntry)

		l := log.With(
			l,
			log.String("key", key.(string)),
			log.String("path", path),
			log.UInt64("size", entry.size),
		)

		if entry.writer != nil {
			err := entry.writer.Close()
			if err != nil {
				l.Error("Failed to close writer on eviction", log.Error(err))
			}
		}

		switch {
		case entry.state == Stored:
			if err := os.Remove(entry.finalPath); err != nil {
				l.Error("Failed to delete main path on eviction", log.Error(err))
				return
			}
		case entry.state == Opened || entry.state == WriteFailed:
			tmpPath := entry.finalPath + TmpSuffix
			if err := os.Remove(tmpPath); err != nil {
				l.Error("Failed to delete tmp path on eviction", log.String("tmp_path", tmpPath), log.Error(err))
				return
			}
		}

	}
}
