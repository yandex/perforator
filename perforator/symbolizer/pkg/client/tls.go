package client

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/yandex/perforator/library/go/certifi"
)

func newTransportCredentialsDialOption(secure bool) (grpc.DialOption, error) {
	creds, err := newTransportCredentials(secure)
	if err != nil {
		return nil, err
	}
	return grpc.WithTransportCredentials(creds), nil
}

func newTransportCredentials(secure bool) (credentials.TransportCredentials, error) {
	if !secure {
		return insecure.NewCredentials(), nil
	}

	certPool, err := certifi.NewCertPool()
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		RootCAs: certPool,
	}), nil
}
