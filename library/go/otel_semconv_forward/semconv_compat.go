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

// Compatibility shims for symbols removed from upstream semconv in v1.40.0.
const (
	ErrorMessageKey               = attribute.Key("error.message")
	RPCMessageCompressedSizeKey   = attribute.Key("rpc.message.compressed_size")
	RPCMessageIDKey               = attribute.Key("rpc.message.id")
	RPCMessageTypeKey             = attribute.Key("rpc.message.type")
	RPCMessageUncompressedSizeKey = attribute.Key("rpc.message.uncompressed_size")
)

var (
	RPCMessageTypeSent     = RPCMessageTypeKey.String("SENT")
	RPCMessageTypeReceived = RPCMessageTypeKey.String("RECEIVED")
)

func ErrorMessage(val string) attribute.KeyValue { return ErrorMessageKey.String(val) }

func RPCMessageCompressedSize(val int) attribute.KeyValue {
	return RPCMessageCompressedSizeKey.Int(val)
}

func RPCMessageID(val int) attribute.KeyValue { return RPCMessageIDKey.Int(val) }

func RPCMessageUncompressedSize(val int) attribute.KeyValue {
	return RPCMessageUncompressedSizeKey.Int(val)
}
