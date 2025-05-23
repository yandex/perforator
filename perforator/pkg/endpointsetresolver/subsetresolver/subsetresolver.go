package subsetresolver

import (
	"math/rand"
	"sync"

	"google.golang.org/grpc/resolver"
)

func NewBuilder(underlying resolver.Builder, subsetsz uint) Builder {
	return Builder{underlying: underlying, subsetsz: subsetsz}
}

type Builder struct {
	underlying resolver.Builder
	subsetsz   uint
}

func (b *Builder) Scheme() string {
	return b.underlying.Scheme()
}

func (b *Builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	return b.underlying.Build(target, &ClientConnWrapper{ClientConn: cc, subsetsz: b.subsetsz, picked: make([]resolver.Endpoint, 0, b.subsetsz)}, opts)
}

type ClientConnWrapper struct {
	resolver.ClientConn
	subsetsz uint
	picked   []resolver.Endpoint
	mu       sync.Mutex
}

// When resolver need to update its state (resolver.State)
// it converts all addresses (soon deprecated api) to endpoints (new api)
// then collects dead endpoints and replaces them
// finally updates resolvers's state
func (c *ClientConnWrapper) UpdateState(state resolver.State) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	endpoints := collectEndpoints(state)
	alive := c.collectAlive(endpoints)
	c.deleteDeadEndpointsFromPicked(alive)
	c.addEndpointsToPicked(endpoints, alive)

	state.Endpoints = c.picked
	state.Addresses = nil
	return c.ClientConn.UpdateState(state)
}

func (c *ClientConnWrapper) collectAlive(endpoints []resolver.Endpoint) map[string]bool {
	alive := make(map[string]bool, c.subsetsz)
	for _, ep := range c.picked {
		if len(ep.Addresses) == 0 {
			continue
		}
		alive[ep.Addresses[0].Addr] = false
	}
	for _, ep := range endpoints {
		if len(ep.Addresses) == 0 {
			continue
		}
		addr := ep.Addresses[0].Addr
		if _, exists := alive[addr]; exists {
			alive[addr] = true
		}
	}
	return alive
}

func (c *ClientConnWrapper) addEndpointsToPicked(endpoints []resolver.Endpoint, alive map[string]bool) {
	rand.Shuffle(len(endpoints), func(i, j int) {
		endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
	})
	for _, candidate := range endpoints {
		if len(c.picked) >= int(c.subsetsz) {
			break
		}

		if len(candidate.Addresses) == 0 {
			continue
		}

		newAddr := candidate.Addresses[0].Addr
		if !alive[newAddr] {
			c.picked = append(c.picked, candidate)
		}
	}
}

func (c *ClientConnWrapper) deleteDeadEndpointsFromPicked(alive map[string]bool) {
	countAlive := 0
	for idx, ep := range c.picked {
		if alive[ep.Addresses[0].Addr] {
			c.picked[countAlive] = c.picked[idx]
			countAlive++
		}
	}
	c.picked = c.picked[:countAlive]
}

func collectEndpoints(state resolver.State) []resolver.Endpoint {
	endpoints := make([]resolver.Endpoint, 0, len(state.Endpoints)+len(state.Addresses))
	endpoints = append(endpoints, state.Endpoints...)
	for _, a := range state.Addresses {
		endpoints = append(endpoints, resolver.Endpoint{
			Addresses: []resolver.Address{a},
		})
	}
	return endpoints
}
