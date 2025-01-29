package certifi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/certifi"
)

func TestInternalCA(t *testing.T) {
	certPool, err := certifi.NewCertPoolInternal()
	require.NoError(t, err)

	caNames, err := getCAsFromPool(certPool)
	require.NoError(t, err)

	expected := []string{
		"YandexInternalRootCA",
		"YandexInternalCA",
	}
	require.ElementsMatch(t, expected, caNames)
}

func TestBundled(t *testing.T) {
	certPool, err := certifi.NewCertPoolBundled()
	require.NoError(t, err)

	caNames, err := getCAsFromPool(certPool)
	require.NoError(t, err)

	expected := []string{
		"YandexInternalRootCA",
		"YandexInternalCA",
		"Certum Trusted Network CA",
		"GlobalSign",
	}
	require.Subset(t, caNames, expected)
}

func TestAuto(t *testing.T) {
	TestBundled(t)
}
