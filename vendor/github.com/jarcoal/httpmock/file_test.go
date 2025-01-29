package httpmock_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

var _ json.Marshaler = httpmock.File("test.json")

func TestFile(t *testing.T) {
	dir, cleanup := tmpDir(t)
	defer cleanup()

	t.Run("Valid JSON file", func(t *testing.T) {
		okFile := filepath.Join(dir, "ok.json")
		writeFile(t, okFile, []byte(`{ "test": true }`))

		encoded, err := json.Marshal(httpmock.File(okFile))
		if err != nil {
			t.Errorf("json.Marshal(%s) failed: %s", okFile, err)
			return
		}
		got, expected := string(encoded), `{"test":true}`
		if got != expected {
			t.Errorf("json.Marshal(%s): got=<%s> expected=<%s>", okFile, got, expected)
		}
	})

	t.Run("Nonexistent JSON file", func(t *testing.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.json")
		_, err := json.Marshal(httpmock.File(nonexistentFile))
		if err == nil {
			t.Errorf("json.Marshal(%s) succeeded, but an error is expected!", nonexistentFile)
		}
	})

	t.Run("Invalid JSON file", func(t *testing.T) {
		badFile := filepath.Join(dir, "bad.json")
		writeFile(t, badFile, []byte(`[123`))

		_, err := json.Marshal(httpmock.File(badFile))
		if err == nil {
			t.Errorf("json.Marshal(%s) succeeded, but an error is expected!", badFile)
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		file := filepath.Join(dir, "ok.raw")
		content := []byte(`abc123`)
		writeFile(t, file, content)

		if got := httpmock.File(file).Bytes(); !bytes.Equal(content, got) {
			t.Errorf("bytes differ:\n      got: %v\n expected: %v", got, content)
		}
	})

	t.Run("Bytes panic", func(t *testing.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.raw")
		panicked, mesg := catchPanic(func() {
			httpmock.File(nonexistentFile).Bytes()
		})
		if !panicked {
			t.Error("No panic detected")
			return
		}
		if !strings.HasPrefix(mesg, "Cannot read "+nonexistentFile) {
			t.Errorf("Bad panic mesg: <%s>", mesg)
		}
	})

	t.Run("String", func(t *testing.T) {
		file := filepath.Join(dir, "ok.txt")
		content := `abc123`
		writeFile(t, file, []byte(content))

		if got := httpmock.File(file).String(); got != content {
			t.Errorf("strings differ:\n      got: <%s>\n expected: <%s>", got, content)
		}
	})

	t.Run("String panic", func(t *testing.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.txt")
		panicked, mesg := catchPanic(func() {
			httpmock.File(nonexistentFile).String() //nolint: govet
		})
		if !panicked {
			t.Error("No panic detected")
			return
		}
		if !strings.HasPrefix(mesg, "Cannot read "+nonexistentFile) {
			t.Errorf("Bad panic mesg: <%s>", mesg)
		}
	})
}
