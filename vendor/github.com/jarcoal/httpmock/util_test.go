package httpmock_test

import (
	"io/ioutil" //nolint: staticcheck
	"net/http"
	"os"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func assertBody(t testing.TB, resp *http.Response, expected string) bool {
	t.Helper()

	require := td.Require(t)
	require.NotNil(resp)

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	require.CmpNoError(err)

	return td.CmpString(t, data, expected)
}

func tmpDir(t testing.TB) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "httpmock")
	td.Require(t).CmpNoError(err)
	return dir, func() { os.RemoveAll(dir) }
}

func writeFile(t testing.TB, file string, content []byte) {
	t.Helper()
	td.Require(t).CmpNoError(ioutil.WriteFile(file, content, 0644))
}
