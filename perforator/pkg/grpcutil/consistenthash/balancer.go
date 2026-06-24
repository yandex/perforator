package consistenthash

import (
	"context"

	"github.com/serialx/hashring"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Name = "consistent_hash"

const ServiceConfig = `{"loadBalancingConfig":[{"consistent_hash":{}}]}`

type keyCtxKey struct{}

func WithKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, keyCtxKey{}, key)
}

func keyFromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(keyCtxKey{}).(string)
	return key, ok
}

func init() {
	balancer.Register(base.NewBalancerBuilder(Name, pickerBuilder{}, base.Config{HealthCheck: true}))
}

type pickerBuilder struct{}

func (pickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	byAddr := make(map[string]balancer.SubConn, len(info.ReadySCs))
	addrs := make([]string, 0, len(info.ReadySCs))
	for sc, sci := range info.ReadySCs {
		addr := sci.Address.Addr
		byAddr[addr] = sc
		addrs = append(addrs, addr)
	}

	return &picker{ring: hashring.New(addrs), byAddr: byAddr}
}

type picker struct {
	ring   *hashring.HashRing
	byAddr map[string]balancer.SubConn
}

func (p *picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	key, ok := keyFromContext(info.Ctx)
	if !ok {
		return balancer.PickResult{}, status.Error(codes.Internal, "consistenthash: no routing key in context")
	}

	addr, ok := p.ring.GetNode(key)
	if !ok {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}
	return balancer.PickResult{SubConn: p.byAddr[addr]}, nil
}
