package httpmock_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock"
)

var _ json.Marshaler = httpmock.File("test.json")

func TestFile(t *testing.T) {
	assert := td.Assert(t)

	dir, cleanup := tmpDir(assert)
	defer cleanup()

	assert.Run("Valid JSON file", func(assert *td.T) {
		okFile := filepath.Join(dir, "ok.json")
		writeFile(assert, okFile, []byte(`{ "test": true }`))

		encoded, err := json.Marshal(httpmock.File(okFile))
		if !assert.CmpNoError(err, "json.Marshal(%s)", okFile) {
			return
		}
		assert.String(encoded, `{"test":true}`)
	})

	assert.Run("Nonexistent JSON file", func(assert *td.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.json")
		_, err := json.Marshal(httpmock.File(nonexistentFile))
		assert.CmpError(err, "json.Marshal(%s), error expected", nonexistentFile)
	})

	assert.Run("Invalid JSON file", func(assert *td.T) {
		badFile := filepath.Join(dir, "bad.json")
		writeFile(assert, badFile, []byte(`[123`))

		_, err := json.Marshal(httpmock.File(badFile))
		assert.CmpError(err, "json.Marshal(%s), error expected", badFile)
	})

	assert.Run("Bytes", func(assert *td.T) {
		file := filepath.Join(dir, "ok.raw")
		content := []byte(`abc123`)
		writeFile(assert, file, content)

		assert.Cmp(httpmock.File(file).Bytes(), content)
	})

	assert.Run("Bytes panic", func(assert *td.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.raw")
		assert.CmpPanic(func() { httpmock.File(nonexistentFile).Bytes() },
			td.HasPrefix("Cannot read "+nonexistentFile))
	})

	assert.Run("String", func(assert *td.T) {
		file := filepath.Join(dir, "ok.txt")
		content := `abc123`
		writeFile(assert, file, []byte(content))

		assert.Cmp(httpmock.File(file).String(), content)
	})

	assert.Run("String panic", func(assert *td.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.txt")
		assert.CmpPanic(
			func() {
				httpmock.File(nonexistentFile).String() //nolint: govet
			},
			td.HasPrefix("Cannot read "+nonexistentFile))
	})
}
