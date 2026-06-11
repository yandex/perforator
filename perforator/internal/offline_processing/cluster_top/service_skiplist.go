package cluster_top

type ServiceSkipList struct {
	services map[string]struct{}
}

func NewServiceSkipList(services []string) *ServiceSkipList {
	m := make(map[string]struct{}, len(services))
	for _, service := range services {
		m[service] = struct{}{}
	}
	return &ServiceSkipList{services: m}
}

func (s *ServiceSkipList) Contains(service string) bool {
	if s == nil || len(s.services) == 0 {
		return false
	}
	_, ok := s.services[service]
	return ok
}
