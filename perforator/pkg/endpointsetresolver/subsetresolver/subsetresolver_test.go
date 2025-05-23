package subsetresolver

import (
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

type mockBuilder struct {
	addrs []resolver.Address
}

func (b *mockBuilder) Scheme() string { return "test" }

func (b *mockBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	return &mockResolver{cc: cc, addrs: b.addrs}, nil
}

type mockResolver struct {
	addrs []resolver.Address
	cc    resolver.ClientConn
}

func (m *mockResolver) ResolveNow(resolver.ResolveNowOptions) {
	state := resolver.State{Addresses: m.addrs}
	err := m.cc.UpdateState(state)
	if err != nil {
		panic(err)
	}
}

func (*mockResolver) Close() {}

type mockClientConn struct {
	mu    sync.Mutex
	state resolver.State
}

func (t *mockClientConn) UpdateState(s resolver.State) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = s
	return nil
}

func (*mockClientConn) ReportError(error) {}

func (*mockClientConn) NewAddress([]resolver.Address) {}

func (*mockClientConn) NewServiceConfig(*serviceconfig.ParseResult) {}

func (*mockClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult {
	return nil
}

func makeAddrs(n, startIdx int) []resolver.Address {
	eps := make([]resolver.Address, n)
	for i := range n {
		addr := fmt.Sprintf("addr-%d", startIdx+i)
		eps[i] = resolver.Address{Addr: addr}
	}
	return eps
}

func checkResolving(t *testing.T, subsetResolver resolver.Resolver, cc *mockClientConn, subsetSize int) {
	subsetResolver.ResolveNow(resolver.ResolveNowOptions{})
	cc.mu.Lock()
	got := cc.state.Endpoints
	cc.mu.Unlock()
	if len(got) != subsetSize {
		t.Fatalf("expected %d endpoints, got %d", subsetSize, len(got))
	}
	orig := map[string]struct{}{}
	for _, addr := range (subsetResolver.(*mockResolver)).addrs {
		orig[addr.Addr] = struct{}{}
	}
	for _, ep := range got {
		addr := ep.Addresses[0].Addr
		if _, ok := orig[addr]; !ok {
			t.Fatalf("wrapper returned unknown address %q", addr)
		}
	}
}

func TestSubsetResolverPicksSubset(t *testing.T) {
	const subsetSize = 5
	const total = 10
	baseBuilder := &mockBuilder{addrs: makeAddrs(total, 0)}
	subsetBuilder := NewBuilder(baseBuilder, subsetSize)
	testCC := &mockClientConn{}

	subsetResolver, err := subsetBuilder.Build(
		resolver.Target{URL: url.URL{Scheme: "test"}},
		testCC,
		resolver.BuildOptions{},
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	checkResolving(t, subsetResolver, testCC, subsetSize)
}

func TestSubsetResolverReplacesDeadEndpoints(t *testing.T) {
	const subsetSize, total = 5, 10
	baseBuilder := &mockBuilder{makeAddrs(total, 0)}
	subsetBuilder := NewBuilder(baseBuilder, subsetSize)
	testCC := &mockClientConn{}

	subsetResolver, err := subsetBuilder.Build(
		resolver.Target{URL: url.URL{Scheme: "test"}},
		testCC,
		resolver.BuildOptions{},
	)

	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	checkResolving(t, subsetResolver, testCC, subsetSize)
	baseResolver := subsetResolver.(*mockResolver)
	baseResolver.addrs = makeAddrs(total, subsetSize+1)
	checkResolving(t, subsetResolver, testCC, subsetSize)
}

func TestMultipleAgents(t *testing.T) {
	const (
		subsetSize                    = 5
		totalServers                  = 100
		agentsNum                     = 10000
		iters                         = 100
		maxAgentPerServerThreshold    = 2000
		maxAgentPerServerThresholdAvg = 700
	)
	agents := make([]resolver.Resolver, agentsNum)
	clientConns := make([]*mockClientConn, agentsNum)
	addrs := makeAddrs(totalServers, 0)
	addrIdx := make(map[string]int, totalServers)
	for i := range totalServers {
		addrIdx[addrs[i].Addr] = i
	}
	for i := range agentsNum {
		baseBuilder := &mockBuilder{addrs: addrs}
		subsetBuilder := NewBuilder(baseBuilder, subsetSize)
		clientConns[i] = &mockClientConn{}
		subsetResolver, err := subsetBuilder.Build(
			resolver.Target{URL: url.URL{Scheme: "test"}},
			clientConns[i],
			resolver.BuildOptions{},
		)

		if err != nil {
			t.Fatalf("build failed: %v", err)
		}
		agents[i] = subsetResolver
	}

	iterCounters := make([]atomic.Uint32, totalServers)
	totalCounters := make([]uint32, totalServers)
	for range iters {
		var wg sync.WaitGroup

		for i, agent := range agents {
			wg.Add(1)
			go func(i int, agent resolver.Resolver) {
				defer wg.Done()
				agent.ResolveNow(resolver.ResolveNowOptions{})
				for _, ep := range clientConns[i].state.Endpoints {
					iterCounters[addrIdx[ep.Addresses[0].Addr]].Add(1)
				}
			}(i, agent)
		}

		wg.Wait()
		for j := range totalServers {
			iterCnt := iterCounters[j].Load()
			if iterCnt > maxAgentPerServerThreshold {
				t.Fatalf("Bad distribution: on one endpoint %d agents", iterCnt)
			}
			totalCounters[j] += iterCnt
			iterCounters[j].Store(0)
		}
	}
	for i := range totalServers {
		avg := float32(totalCounters[i]) / iters
		if avg > maxAgentPerServerThresholdAvg {
			t.Fatalf("Too big per server avg agents number: %f", avg)
		}
	}
}
