package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePassword(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		password, err := resolvePassword("secret", "")
		require.NoError(t, err)
		require.Equal(t, "secret", password)
	})

	t.Run("file", func(t *testing.T) {
		path := writePasswordFile(t, "secret\n")

		password, err := resolvePassword("", path)
		require.NoError(t, err)
		require.Equal(t, "secret", password)
	})

	t.Run("file with CRLF", func(t *testing.T) {
		path := writePasswordFile(t, "secret\r\n")

		password, err := resolvePassword("", path)
		require.NoError(t, err)
		require.Equal(t, "secret", password)
	})

	t.Run("mutually exclusive", func(t *testing.T) {
		password, err := resolvePassword("inline-secret", "secret-file")
		require.Empty(t, password)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "inline-secret")
	})

	t.Run("missing file", func(t *testing.T) {
		password, err := resolvePassword("", filepath.Join(t.TempDir(), "missing"))
		require.Empty(t, password)
		require.Error(t, err)
		require.False(t, strings.Contains(err.Error(), "password="))
	})
}

func writePasswordFile(t *testing.T, password string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "password")
	require.NoError(t, os.WriteFile(path, []byte(password), 0o600))
	return path
}
