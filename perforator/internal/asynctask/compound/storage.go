package compound

import (
	"errors"
	"fmt"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	inmemorytaskservice "github.com/yandex/perforator/perforator/internal/asynctask/inmemory"
	postgrestaskservice "github.com/yandex/perforator/perforator/internal/asynctask/postgres"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrUnspecifiedTasksService = errors.New("unspecified asynctask service")
)

func NewTasksService(logger xlog.Logger, reg metrics.Registry, opts ...Option) (asynctask.TaskService, error) {
	options := defaultOpts()
	for _, applyOpt := range opts {
		applyOpt(options)
	}

	switch {
	case options.inMemoryConfig != nil:
		tasks, err := inmemorytaskservice.NewInMemoryTaskService(options.inMemoryConfig, logger, reg)
		if err != nil {
			return nil, fmt.Errorf("failed to create asynctask service: %w", err)
		}

		return tasks, nil
	case options.postgresCluster != nil && options.postgresConfig != nil:
		tasks, err := postgrestaskservice.NewTaskService(options.postgresConfig, options.postgresCluster, logger, reg)
		if err != nil {
			return nil, fmt.Errorf("failed to create asynctask service: %w", err)
		}

		return tasks, nil
	default:
		return nil, ErrUnspecifiedTasksService
	}
}
