package server

import (
	"context"
	"sync"
	"time"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/databases"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type BannedUsersRegistry struct {
	log     xlog.Logger
	cluster *hasql.Cluster

	usersmu         sync.RWMutex
	users           map[string]bool
	usersLastUpdate time.Time

	bannedUserCount    metrics.IntGauge
	bannedUserCacheAge metrics.FuncGauge
}

func NewBannedUsersRegistry(ctx context.Context, log xlog.Logger, metrics metrics.Registry, db *databases.Databases) (registry *BannedUsersRegistry) {
	registry = &BannedUsersRegistry{
		log:             log.WithName("BannedUsers"),
		cluster:         db.PostgresCluster,
		users:           make(map[string]bool),
		usersLastUpdate: time.Now(),
	}

	registry.instrument(metrics)

	go registry.runPoller(ctx)

	return registry
}

func (registry *BannedUsersRegistry) instrument(metrics metrics.Registry) {
	registry.bannedUserCount = metrics.IntGauge("banned_user.count")
	registry.bannedUserCacheAge = metrics.FuncGauge("banned_user.cache_age.seconds", registry.calcUsersCacheAge)
}

func (registry *BannedUsersRegistry) calcUsersCacheAge() float64 {
	var lastUpdate time.Time
	registry.usersmu.RLock()
	lastUpdate = registry.usersLastUpdate
	registry.usersmu.RUnlock()

	return time.Since(lastUpdate).Seconds()
}

func (registry *BannedUsersRegistry) IsBanned(user string) bool {
	registry.usersmu.RLock()
	defer registry.usersmu.RUnlock()
	return registry.users[user]
}

func (registry *BannedUsersRegistry) runPoller(ctx context.Context) {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
		}

		registry.update(ctx)
	}
}

func (registry *BannedUsersRegistry) update(ctx context.Context) {
	db := registry.cluster.StandbyPreferred().DBx()

	var logins []string
	err := db.Select(&logins, "SELECT login FROM banned_users")
	if err != nil {
		registry.log.Error(ctx, "Failed to load banned users", log.Error(err))
		return
	}
	registry.log.Debug(ctx, "Loaded banned users", log.Int("count", len(logins)))

	users := make(map[string]bool, len(logins))
	for _, login := range logins {
		users[login] = true
	}

	registry.usersmu.Lock()
	registry.users = users
	registry.usersLastUpdate = time.Now()
	registry.usersmu.Unlock()

	registry.bannedUserCount.Set(int64(len(logins)))
}
