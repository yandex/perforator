package process

import (
	"fmt"
	"os"
	"time"

	"github.com/karlseguin/ccache/v3"

	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xelf"
)

type BuildIDKey struct {
	Device procfs.Device
	Inode  procfs.Inode
}

type BuildIDCache struct {
	cache *ccache.Cache[*xelf.BuildInfo]
}

func NewBuildIDCache() *BuildIDCache {
	return &BuildIDCache{cache: ccache.New[*xelf.BuildInfo](ccache.Configure[*xelf.BuildInfo]())}
}

func (c *BuildIDCache) Load(key BuildIDKey, f *os.File) (*xelf.BuildInfo, error) {
	s := fmt.Sprintf("%d/%d/%d/%d", key.Device.Maj, key.Device.Min, key.Inode.ID, key.Inode.Gen)

	item, err := c.cache.Fetch(s, time.Hour, func() (*xelf.BuildInfo, error) {
		return xelf.ReadBuildInfo(f)
	})

	if err != nil {
		return nil, err
	}

	return item.Value(), nil
}
