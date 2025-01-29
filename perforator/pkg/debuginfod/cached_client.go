package debuginfod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yandex/perforator/perforator/pkg/atomicfs"
	"github.com/yandex/perforator/perforator/pkg/xelf"
)

////////////////////////////////////////////////////////////////////////////////

type CachedClientOption func(c *CachedClient) error

func WithCacheDir(dir string) CachedClientOption {
	return func(c *CachedClient) error {
		c.cacheDir = dir
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////

type CachedClient struct {
	client   *Client
	cacheDir string
}

func NewCachedClient(opts ...any) (*CachedClient, error) {
	clientOpts := make([]ClientOption, 0)
	cacheOpts := make([]CachedClientOption, 0)
	for _, opt := range opts {
		switch v := any(opt).(type) {
		case ClientOption:
			clientOpts = append(clientOpts, v)
		case CachedClientOption:
			cacheOpts = append(cacheOpts, v)
		default:
			return nil, fmt.Errorf("unexpected cached client option %T", v)
		}
	}

	client, err := NewClient(clientOpts...)
	if err != nil {
		return nil, err
	}

	cachedClient := &CachedClient{client: client}
	for _, opt := range cacheOpts {
		opt(cachedClient)
	}

	if cachedClient.cacheDir == "" {
		cachedClient.cacheDir, err = defaultCacheDir()
		if err != nil {
			return nil, err
		}
	}

	return cachedClient, nil
}

func defaultCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "debuginfod"), nil
}

func (c *CachedClient) OpenExecutable(ctx context.Context, buildID string) (*os.File, error) {
	f, err := c.fetch(func(f *atomicfs.File) error {
		_, err := c.client.FetchExecutable(ctx, buildID, f)
		return err
	}, buildID, "executable")
	if err != nil {
		return nil, err
	}

	err = c.validateFetchedDebugInfo(buildID, f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return f, nil
}

func (c *CachedClient) OpenDebugInfo(ctx context.Context, buildID string) (*os.File, error) {
	f, err := c.fetch(func(f *atomicfs.File) error {
		_, err := c.client.FetchDebugInfo(ctx, buildID, f)
		return err
	}, buildID, "debuginfo")
	if err != nil {
		return nil, err
	}

	err = c.validateFetchedDebugInfo(buildID, f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	return f, nil
}

func (c *CachedClient) OpenSection(ctx context.Context, buildID, section string) (*os.File, error) {
	return c.fetch(func(f *atomicfs.File) error {
		_, err := c.client.FetchSection(ctx, buildID, section, f)
		return err
	}, buildID, fmt.Sprintf("section-%s", section))
}

func (c *CachedClient) validateFetchedDebugInfo(expectedBuildID string, f *os.File) error {
	realBuildID, err := xelf.ReadBuildID(f)
	if err != nil {
		return fmt.Errorf("failed to read build id: %w", err)
	}

	if realBuildID != expectedBuildID {
		return fmt.Errorf("got malformed executable from a debuginfod server: expected build id %q, got build id %q", expectedBuildID, realBuildID)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (c *CachedClient) path(parts ...string) string {
	return filepath.Join(c.cacheDir, filepath.Join(parts...))
}

func (c *CachedClient) fetch(populate func(f *atomicfs.File) error, parts ...string) (*os.File, error) {
	path := c.path(parts...)

	f, err := c.get(path)
	if err == nil {
		return f, nil
	}

	return c.put(path, populate)
}

func (c *CachedClient) get(path string) (*os.File, error) {
	return os.Open(path)
}

func (c *CachedClient) put(path string, populate func(f *atomicfs.File) error) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(path), 0o777)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare cached executable directory: %w", err)
	}

	f, err := atomicfs.Create(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Discard()
	}()

	err = populate(f)
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	return c.get(path)
}

////////////////////////////////////////////////////////////////////////////////
