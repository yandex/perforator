package postgres

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type SSLMode = string

const (
	SSLModeDisable    SSLMode = "disable"
	SSLModeAllow      SSLMode = "allow"
	SSLModePrefer     SSLMode = "prefer"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCA   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
)

var (
	sslModes = map[SSLMode]struct{}{
		SSLModeDisable:    {},
		SSLModeAllow:      {},
		SSLModePrefer:     {},
		SSLModeRequire:    {},
		SSLModeVerifyCA:   {},
		SSLModeVerifyFull: {},
	}
)

type Endpoint struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

func (e *Endpoint) Addr() string {
	if e.Port == 0 {
		return e.Host
	}

	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

type AuthConfig struct {
	User        string `yaml:"user"`
	PasswordEnv string `yaml:"password_env"`
}

type Config struct {
	AuthConfig  AuthConfig `yaml:"auth"`
	DB          string     `yaml:"db"`
	Endpoints   []Endpoint `yaml:"endpoints"`
	SSLMode     SSLMode    `yaml:"sslmode,omitempty"`
	SSLRootCert string     `yaml:"sslrootcert,omitempty"`
}

func ConnectionString(auth *AuthConfig, db string, endpoint *Endpoint, sslMode SSLMode, sslRootCert string) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(
		"postgresql://%s@%s/%s?",
		url.UserPassword(auth.User, os.Getenv(auth.PasswordEnv)).String(),
		endpoint.Addr(),
		db,
	))

	if sslMode == "" {
		sslMode = SSLModeRequire
	}

	if _, ok := sslModes[sslMode]; !ok {
		return "", fmt.Errorf("unknown sslmode for postgresql db connection: your value \"%s\"", sslMode)
	}

	b.WriteString(fmt.Sprintf("sslmode=%s", sslMode))

	if sslRootCert != "" {
		b.WriteString(fmt.Sprintf("&sslrootcert=%s", sslRootCert))
	}

	return b.String(), nil
}
