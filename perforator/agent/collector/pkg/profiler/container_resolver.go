package profiler

// ContainerResolver identifies a container by its pod cgroup and its own cgroup basename.
type ContainerResolver interface {
	// parentCgroupPath is relative to the local cgroup mount.
	ResolveContainer(parentCgroupPath, containerCgroupBaseName string) (podServiceName, containerName string, found bool)
}

func WithContainerResolver(resolver ContainerResolver) Option {
	return func(p *Profiler) error {
		p.containerResolver = resolver
		return nil
	}
}
