package certifi

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientTLSConfig enables TLS configuration for the client
type ClientTLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CAFile             string `yaml:"ca_file_path"`
	CertFile           string `yaml:"certificate_file_path"`
	KeyFile            string `yaml:"key_file_path"`
	ServerNameOverride string `yaml:"server_name_override,omitempty"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify,omitempty"`
}

// ServerTLSConfig enables TLS configuration for the server
type ServerTLSConfig struct {
	Enabled      bool   `yaml:"enabled"`
	CAFile       string `yaml:"ca_file_path"`
	CertFile     string `yaml:"certificate_file_path"`
	KeyFile      string `yaml:"key_file_path"`
	VerifyClient bool   `yaml:"verify_client"`

	CertificateFileDeprecated string `yaml:"certificate_file"`
	KeyFileDeprecated         string `yaml:"key_file"`
}

// BuildTLSConfig builds a golang tls.Config from its fields for client,
// It returns nil config if TLS is not enabled.
//
// Note that while a nil *tls.Config and an empty &tls.Config{} are mostly interchangeable,
// some libraries specifically require a nil config to disable TLS.
// For example, see the ClickHouse driver documentation:
// https://pkg.go.dev/github.com/ClickHouse/clickhouse-go/v2#readme-tls-ssl
func (c *ClientTLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if c == nil || !c.Enabled {
		return nil, nil
	}

	caCertPool, err := CertPoolFromFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate pool: %w", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	// Load client certificates if provided
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if c.ServerNameOverride != "" {
		tlsConfig.ServerName = c.ServerNameOverride
	}

	return tlsConfig, nil
}

// GRPCDialOptions returns gRPC dial options with appropriate TLS credentials for client
func (c *ClientTLSConfig) GRPCDialOptions() ([]grpc.DialOption, error) {
	var opts []grpc.DialOption

	if c == nil || !c.Enabled {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		return opts, nil
	}

	tlsConfig, err := c.BuildTLSConfig()
	if err != nil {
		return nil, err
	}

	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	return opts, nil
}

// HTTPClient returns a golang http.client configured with the appropriate TLS settings
func (c *ClientTLSConfig) HTTPClient() (*http.Client, error) {
	if c == nil || !c.Enabled {
		return &http.Client{}, nil
	}

	tlsConfig, err := c.BuildTLSConfig()
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// BuildTLSConfig builds a golang tls.Config from its fields for server,
// It returns nil config if TLS is not enabled.
//
// Note that while a nil *tls.Config and an empty &tls.Config{} are mostly interchangeable,
// some libraries specifically require a nil config to disable TLS.
// For example, see the ClickHouse driver documentation:
// https://pkg.go.dev/github.com/ClickHouse/clickhouse-go/v2#readme-tls-ssl
func (c *ServerTLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if c == nil || !c.Enabled {
		return nil, nil
	}

	if c.CertFile == "" || c.KeyFile == "" {
		return nil, fmt.Errorf("server TLS config requires certificate and key files")
	}

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	caCertPool, err := CertPoolFromFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate pool: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
	}

	if c.VerifyClient {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// GRPCServerOptions returns gRPC server options with appropriate TLS credentials
func (c *ServerTLSConfig) GRPCServerOptions() ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	if c == nil || !c.Enabled {
		return opts, nil
	}

	tlsConfig, err := c.BuildTLSConfig()
	if err != nil {
		return nil, err
	}

	opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	return opts, nil
}
