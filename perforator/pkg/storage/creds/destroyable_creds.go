package creds

import (
	"google.golang.org/grpc/credentials"
)

type DestroyablePerRPCCredentials interface {
	credentials.PerRPCCredentials
	Destroy()
}
