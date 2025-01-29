package certifi

import "crypto/x509"

func NewDefaultCertPool() (*x509.CertPool, error) {
	return x509.SystemCertPool()
}
