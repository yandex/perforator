package httpmock_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func assertBody(t *testing.T, resp *http.Response, expected string) bool {
	defer resp.Body.Close()

	helper(t).Helper()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)

	if got != expected {
		t.Errorf("Got body: %#v, expected: %#v", got, expected)
		return false
	}
	return true
}

// Stolen from https://github.com/maxatome/go-testdeep
func catchPanic(fn func()) (panicked bool, ret string) {
	func() {
		defer func() {
			panicParam := recover()
			if panicked {
				ret = fmt.Sprint(panicParam)
			}
		}()
		panicked = true
		fn()
		panicked = false
	}()
	return
}

func tmpDir(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "httpmock")
	if err != nil {
		helper(t).Helper()
		t.Fatal(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

func writeFile(t *testing.T, file string, content []byte) {
	err := ioutil.WriteFile(file, content, 0644)
	if err != nil {
		helper(t).Helper()
		t.Fatal(err)
	}
}

// fakeHelper allows to compensate the absence of
// (*testing.T).Helper() in go<1.9.
type fakeHelper struct{}

func (f fakeHelper) Helper() {}

type helperAble interface {
	Helper()
}

func helper(t interface{}) helperAble {
	if th, ok := t.(helperAble); ok {
		return th
	}
	return fakeHelper{}
}
