package models

type ParallelDownloadConfig struct {
	Concurrency int
	PartSize    int64
}
