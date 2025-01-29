package s3

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	defaultRegion       = "us-east-1"
	defaultAccessKeyEnv = "S3_ACCESS_KEY"
	defaultSecretKeyEnv = "S3_SECRET_KEY"
	defaultMaxRetries   = 10
)

type Config struct {
	Endpoint string `yaml:"endpoint"`

	SecretKeyPath string `yaml:"secret_key_path"`
	AccessKeyPath string `yaml:"access_key_path"`
	SecretKeyEnv  string `yaml:"secret_key_env"`
	AccessKeyEnv  string `yaml:"access_key_env"`

	Region             string `yaml:"region"`
	ForcePathStyle     *bool  `yaml:"force_path_style"`
	InsecureSkipVerify bool   `yaml:"insecure,omitempty"`
	CACertPath         string `yaml:"ca_cert_path,omitempty"`
}

func (c *Config) fillDefault() {
	if c.Region == "" {
		c.Region = defaultRegion
	}
	if c.AccessKeyEnv == "" && c.AccessKeyPath == "" {
		c.AccessKeyEnv = defaultAccessKeyEnv
	}
	if c.SecretKeyEnv == "" && c.SecretKeyPath == "" {
		c.SecretKeyEnv = defaultSecretKeyEnv
	}
}

func loadKey(path, env string) (string, error) {
	if path != "" {
		value, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read secret key from %s: %w", path, err)
		}

		return string(value), nil
	}

	if env != "" {
		value, ok := os.LookupEnv(env)
		if !ok {
			return "", fmt.Errorf("environment variable %s is not set", env)
		}

		return value, nil
	}

	return "", fmt.Errorf("no key path or environment variable provided")
}

func NewClient(c *Config) (*s3.S3, error) {
	c.fillDefault()

	secretKey, err := loadKey(c.SecretKeyPath, c.SecretKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret key: %w", err)
	}

	accessKey, err := loadKey(c.AccessKeyPath, c.AccessKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to read access key: %w", err)
	}

	config := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
		WithEndpoint(c.Endpoint).
		WithRegion(c.Region).
		WithMaxRetries(defaultMaxRetries)

	if c.ForcePathStyle != nil {
		config = config.WithS3ForcePathStyle(*c.ForcePathStyle)
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
	if !c.InsecureSkipVerify && c.CACertPath != "" {
		cert, err := os.ReadFile(c.CACertPath)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(cert) {
			return nil, fmt.Errorf("failed to add server CA's certificate, path: %s", c.CACertPath)
		}

		tlsConf.RootCAs = certPool
	}

	config = config.WithHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
		},
	})

	session, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	return s3.New(session), nil
}
