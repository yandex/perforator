package deploysystemmodel

import "context"

type Container interface {
	Name() string
	CgroupBaseName() string
}

type Pod interface {
	ID() string
	Topology() string
	Labels() map[string]string
	CgroupName() string
	Containers() []Container
	ServiceName() string
	IsPerforatorEnabled() (*bool, string)
}

type PodsLister interface {
	Init(ctx context.Context) error
	List() ([]Pod, error)
	GetHost() string
}
