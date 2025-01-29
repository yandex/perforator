package driver

type PrepareBatchOptions struct {
	ReleaseConnection bool
}

type PrepareBatchOption func(options *PrepareBatchOptions)

func WithReleaseConnection() PrepareBatchOption {
	return func(options *PrepareBatchOptions) {
		options.ReleaseConnection = true
	}
}
