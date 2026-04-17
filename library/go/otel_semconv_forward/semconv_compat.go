// Compatibility shims for symbols removed from upstream semconv in v1.39.0.
// Kept local so that bumping the forward package does not force codemods in users.
// See: go.opentelemetry.io/otel/semconv/v1.39.0/MIGRATION.md.

package semconv

import "go.opentelemetry.io/otel/attribute"

const (
	PeerServiceKey       = attribute.Key("peer.service")
	RPCServiceKey        = attribute.Key("rpc.service")
	RPCSystemKey         = attribute.Key("rpc.system")
	RPCGRPCStatusCodeKey = attribute.Key("rpc.grpc.status_code")
)

var RPCSystemGRPC = RPCSystemKey.String("grpc")

func PeerService(val string) attribute.KeyValue { return PeerServiceKey.String(val) }
func RPCService(val string) attribute.KeyValue  { return RPCServiceKey.String(val) }
