package debuginfod

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

// Implementation of the debuginfod client.
// See https://www.mankier.com/8/debuginfod#Webapi for details.
type Client struct {
	r    *resty.Client
	l    xlog.Logger
	urls []string
}

////////////////////////////////////////////////////////////////////////////////

type ClientOption func(c *Client) error

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		c.r.SetTimeout(timeout)
		return nil
	}
}

func WithRetryCount(count int) ClientOption {
	return func(c *Client) error {
		c.r.SetRetryCount(count)
		return nil
	}
}

func WithLogger(logger xlog.Logger) ClientOption {
	return func(c *Client) error {
		logger := logger.WithName("debuginfod")
		c.r.SetLogger(logger.WithName("resty").Fmt())
		c.l = logger
		return nil
	}
}

func WithURL(url string) ClientOption {
	return func(c *Client) error {
		c.addURL(url)
		return nil
	}
}

func WithURLs(urls ...string) ClientOption {
	return func(c *Client) error {
		for _, url := range urls {
			if url := strings.TrimSpace(url); url != "" {
				c.addURL(url)
			}
		}
		return nil
	}
}

func WithEnvConfig() ClientOption {
	return func(c *Client) error {
		env := os.Getenv("DEBUGINFOD_URLS")
		urls := strings.Split(env, ",")
		return WithURLs(urls...)(c)
	}
}

func WithHTTPClientOption(opt func(c *resty.Client) error) ClientOption {
	return func(c *Client) error {
		return opt(c.r)
	}
}

////////////////////////////////////////////////////////////////////////////////

const (
	defaultTimeout    = time.Hour
	defaultRetryCount = 3
)

func NewClient(opts ...ClientOption) (*Client, error) {
	r := resty.New().
		SetTimeout(defaultTimeout).
		SetRetryCount(defaultRetryCount)

	c := &Client{r: r, l: xlog.NewNop()}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if len(c.urls) == 0 {
		return nil, ErrNoEndpoints
	}

	return c, nil
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) FetchDebugInfo(ctx context.Context, buildID string, w io.Writer) (int64, error) {
	return c.downloadFile(ctx, fmt.Sprintf(`/buildid/%s/debuginfo`, buildID), w)
}

func (c *Client) FetchExecutable(ctx context.Context, buildID string, w io.Writer) (int64, error) {
	return c.downloadFile(ctx, fmt.Sprintf(`/buildid/%s/executable`, buildID), w)
}

func (c *Client) FetchSection(ctx context.Context, buildID, section string, w io.Writer) (int64, error) {
	return c.downloadFile(ctx, fmt.Sprintf(`/buildid/%s/section/%s`, buildID, section), w)
}

////////////////////////////////////////////////////////////////////////////////

func (c *Client) downloadFile(ctx context.Context, file string, w io.Writer) (count int64, err error) {
	ctx = xlog.WrapContext(ctx, log.String("debuginfod.file", file))
	defer func() {
		if err != nil {
			c.l.Warn(ctx, "Failed to fetch file from debuginfod server", log.Error(err))
		} else {
			c.l.Info(ctx, "Fetched file from debuginfod server", log.Int64("size", count))
		}
	}()

	var r io.ReadCloser

	err = c.tryEachURL(func(endpoint string) error {
		url := endpoint + file

		res, err := c.r.R().
			SetDoNotParseResponse(true).
			SetContext(ctx).
			Get(url)
		if err != nil {
			return err
		}

		if res.IsError() {
			return fmt.Errorf("request to %s failed, status: %s", url, res.Status())
		}

		r = res.RawBody()
		return nil
	})
	if err != nil {
		return 0, err
	}
	defer r.Close()

	return io.Copy(w, r)
}

var ErrNoEndpoints = errors.New("no debuginfod urls set")

func (c *Client) tryEachURL(do func(url string) error) error {
	if len(c.urls) < 1 {
		return ErrNoEndpoints
	}

	var errs []error

	for _, url := range c.urls {
		err := do(url)
		if err == nil {
			return nil
		}
		errs = append(errs, fmt.Errorf("server %s failed: %w", url, err))
	}

	return errors.Join(errs...)
}

func (c *Client) addURL(url string) {
	c.urls = append(c.urls, strings.TrimSuffix(url, "/"))
}

////////////////////////////////////////////////////////////////////////////////
