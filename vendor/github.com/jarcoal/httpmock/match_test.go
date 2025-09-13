package httpmock_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil" //nolint: staticcheck
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock"
	"github.com/jarcoal/httpmock/internal"
)

func TestMatcherFunc_AndOr(t *testing.T) {
	ok := httpmock.MatcherFunc(func(*http.Request) bool { return true })
	bad := httpmock.MatcherFunc(func(*http.Request) bool { return false })

	td.CmpTrue(t, ok(nil))
	td.CmpFalse(t, bad(nil))

	t.Run("Or", func(t *testing.T) {
		td.CmpTrue(t, ok.Or(bad).Or(bad).Or(bad)(nil))
		td.CmpTrue(t, bad.Or(bad).Or(bad).Or(ok)(nil))
		td.CmpFalse(t, bad.Or(bad).Or(bad).Or(bad)(nil))
		td.CmpNil(t, bad.Or(bad).Or(bad).Or(nil))
		td.CmpNil(t, (httpmock.MatcherFunc)(nil).Or(bad).Or(bad).Or(bad))
		td.CmpTrue(t, ok.Or()(nil))
	})

	t.Run("And", func(t *testing.T) {
		td.CmpTrue(t, ok.And(ok).And(ok).And(ok)(nil))
		td.CmpTrue(t, ok.And(ok).And(nil).And(ok)(nil))
		td.CmpFalse(t, ok.And(ok).And(bad).And(ok)(nil))
		td.CmpFalse(t, bad.And(ok).And(ok).And(nil)(nil))
		td.CmpTrue(t, ok.And()(nil))
		td.CmpTrue(t, ok.And(nil)(nil))
		td.CmpNil(t, (httpmock.MatcherFunc)(nil).And(nil).And(nil).And(nil))
		td.CmpTrue(t, (httpmock.MatcherFunc)(nil).And(ok)(nil))
	})
}

func TestMatcherFunc_Check(t *testing.T) {
	ok := httpmock.MatcherFunc(func(*http.Request) bool { return true })
	bad := httpmock.MatcherFunc(func(*http.Request) bool { return false })

	td.CmpTrue(t, ok.Check(nil))
	td.CmpTrue(t, (httpmock.MatcherFunc)(nil).Check(nil))
	td.CmpFalse(t, bad.Check(nil))
}

func TestNewMatcher(t *testing.T) {
	autogenName := td.Re(`^~[0-9a-f]{10} @.*/httpmock_test\.TestNewMatcher.*/match_test.go:\d+\z`)

	t.Run("NewMatcher", func(t *testing.T) {
		td.Cmp(t,
			httpmock.NewMatcher("xxx", func(*http.Request) bool { return true }),
			td.Struct(httpmock.Matcher{}, td.StructFields{
				"name": "xxx",
				"fn":   td.NotNil(),
			}))

		td.Cmp(t, httpmock.NewMatcher("", nil), httpmock.Matcher{})

		td.Cmp(t, httpmock.NewMatcher("", func(*http.Request) bool { return true }),
			td.Struct(httpmock.Matcher{}, td.StructFields{
				"name": autogenName,
				"fn":   td.NotNil(),
			}))
	})

	req := func(t testing.TB, body string, header ...string) *http.Request {
		req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		td.Require(t).CmpNoError(err)
		req.Header.Set("Content-Type", "text/plain")
		for i := 0; i < len(header)-1; i += 2 {
			req.Header.Set(header[i], header[i+1])
		}
		return req
	}

	reqCopyBody := func(t testing.TB, body string, header ...string) *http.Request {
		req := req(t, body, header...)
		req.Body = httpmock.NewBodyCopyOnRead(req.Body)
		return req
	}

	t.Run("BodyContainsBytes", func(t *testing.T) {
		m := httpmock.BodyContainsBytes([]byte("ip"))
		td.Cmp(t, m.Name(), autogenName)
		td.CmpTrue(t, m.Check(req(t, "pipo")))
		td.CmpFalse(t, m.Check(req(t, "bingo")))

		td.CmpTrue(t, m.Check(reqCopyBody(t, "pipo")))
		td.CmpFalse(t, m.Check(reqCopyBody(t, "bingo")))
	})

	t.Run("BodyContainsString", func(t *testing.T) {
		m := httpmock.BodyContainsString("ip")
		td.Cmp(t, m.Name(), autogenName)
		td.CmpTrue(t, m.Check(req(t, "pipo")))
		td.CmpFalse(t, m.Check(req(t, "bingo")))

		td.CmpTrue(t, m.Check(reqCopyBody(t, "pipo")))
		td.CmpFalse(t, m.Check(reqCopyBody(t, "bingo")))
	})

	t.Run("HeaderExists", func(t *testing.T) {
		m := httpmock.HeaderExists("X-Custom")
		td.Cmp(t, m.Name(), autogenName)
		td.CmpTrue(t, m.Check(req(t, "pipo", "X-Custom", "zzz")))
		td.CmpFalse(t, m.Check(req(t, "bingo")))
	})

	t.Run("HeaderIs", func(t *testing.T) {
		m := httpmock.HeaderIs("X-Custom", "zzz")
		td.Cmp(t, m.Name(), autogenName)
		td.CmpTrue(t, m.Check(req(t, "pipo", "X-Custom", "zzz")))
		td.CmpFalse(t, m.Check(req(t, "bingo", "X-Custom", "aaa")))
		td.CmpFalse(t, m.Check(req(t, "bingo")))
	})

	t.Run("HeaderContains", func(t *testing.T) {
		m := httpmock.HeaderContains("X-Custom", "zzz")
		td.Cmp(t, m.Name(), autogenName)
		td.CmpTrue(t, m.Check(req(t, "pipo", "X-Custom", "aaa zzz bbb")))
		td.CmpFalse(t, m.Check(req(t, "bingo")))
	})
}

func TestMatcher_NameWithName(t *testing.T) {
	autogenName := td.Re(`^~[0-9a-f]{10} @.*/httpmock_test\.TestMatcher_NameWithName.*/match_test.go:\d+\z`)

	t.Run("default", func(t *testing.T) {
		m := httpmock.NewMatcher("", nil)
		td.Cmp(t, m.Name(), "", "no autogen for nil fn (= default)")

		td.Cmp(t, m.WithName("pipo").Name(), "pipo")
		td.Cmp(t, m.Name(), "", "original Matcher stay untouched")

		td.Cmp(t, m.WithName("pipo").WithName("").Name(), "", "no autogen for nil fn")
	})

	t.Run("non-default", func(t *testing.T) {
		m := httpmock.NewMatcher("xxx", func(*http.Request) bool { return true })
		td.Cmp(t, m.Name(), "xxx")

		td.Cmp(t, m.WithName("pipo").Name(), "pipo")
		td.Cmp(t, m.Name(), "xxx", "original Matcher stay untouched")

		td.Cmp(t, m.WithName("pipo").WithName("").Name(), autogenName)
	})
}

func TestMatcher_AndOr(t *testing.T) {
	ok := httpmock.MatcherFunc(func(*http.Request) bool { return true })
	bad := httpmock.MatcherFunc(func(*http.Request) bool { return false })

	t.Run("Or", func(t *testing.T) {
		m := httpmock.NewMatcher("a", ok).
			Or(httpmock.NewMatcher("b", bad)).
			Or(httpmock.NewMatcher("c", ok))
		td.Cmp(t, m.Name(), "a")
		td.CmpTrue(t, m.Check(nil))

		m = httpmock.NewMatcher("a", ok).
			Or(httpmock.NewMatcher("", nil)).
			Or(httpmock.NewMatcher("c", ok))
		td.Cmp(t, m.Name(), "")
		td.CmpZero(t, m.FnPointer())

		m = httpmock.NewMatcher("a", ok).Or()
		td.Cmp(t, m.Name(), "a")
		td.CmpTrue(t, m.Check(nil))

		m = httpmock.NewMatcher("a", bad).
			Or(httpmock.NewMatcher("b", bad)).
			Or(httpmock.NewMatcher("c", ok))
		td.Cmp(t, m.Name(), "a")
		td.CmpTrue(t, m.Check(nil))

		m = httpmock.NewMatcher("a", bad).
			Or(httpmock.NewMatcher("b", bad)).
			Or(httpmock.NewMatcher("c", bad))
		td.Cmp(t, m.Name(), "a")
		td.CmpFalse(t, m.Check(nil))
	})

	t.Run("And", func(t *testing.T) {
		m := httpmock.NewMatcher("a", ok).
			And(httpmock.NewMatcher("b", ok)).
			And(httpmock.NewMatcher("c", ok))
		td.Cmp(t, m.Name(), "a")
		td.CmpTrue(t, m.Check(nil))

		m = httpmock.NewMatcher("a", ok).
			And(httpmock.NewMatcher("b", bad)).
			And(httpmock.NewMatcher("c", ok))
		td.Cmp(t, m.Name(), "a")
		td.CmpFalse(t, m.Check(nil))

		mInit := httpmock.NewMatcher("", nil)
		m = mInit.And(httpmock.NewMatcher("", nil)).
			And(httpmock.NewMatcher("", nil))
		td.Cmp(t, m.Name(), mInit.Name())
		td.CmpZero(t, m.FnPointer())

		m = httpmock.NewMatcher("a", ok).And()
		td.Cmp(t, m.Name(), "a")
		td.CmpTrue(t, m.Check(nil))
	})
}

var matchers = []httpmock.MatcherFunc{
	func(*http.Request) bool { return false },
	func(*http.Request) bool { return true },
}

func findMatcher(fnPtr uintptr) int {
	if fnPtr == 0 {
		return -1
	}
	for i, gm := range matchers {
		if fnPtr == reflect.ValueOf(gm).Pointer() {
			return i
		}
	}
	return -2
}

func newMR(name string, num int) httpmock.MatchResponder {
	if num < 0 {
		// default matcher
		return httpmock.NewMatchResponder(httpmock.NewMatcher(name, nil), nil)
	}
	return httpmock.NewMatchResponder(httpmock.NewMatcher(name, matchers[num]), nil)
}

func checkMRs(t testing.TB, mrs httpmock.MatchResponders, names ...string) {
	td.Cmp(t, mrs, td.Smuggle(
		func(mrs httpmock.MatchResponders) []string {
			var ns []string
			for _, mr := range mrs {
				ns = append(ns, fmt.Sprintf("%s:%d",
					mr.Matcher().Name(), findMatcher(mr.Matcher().FnPointer())))
			}
			return ns
		},
		names))
}

func TestMatchResponders_add_remove(t *testing.T) {
	var mrs httpmock.MatchResponders
	mrs = mrs.Add(newMR("foo", 0))
	mrs = mrs.Add(newMR("bar", 0))
	checkMRs(t, mrs, "bar:0", "foo:0")
	mrs = mrs.Add(newMR("bar", 1))
	mrs = mrs.Add(newMR("", -1))
	checkMRs(t, mrs, "bar:1", "foo:0", ":-1")

	mrs = mrs.Remove("foo")
	checkMRs(t, mrs, "bar:1", ":-1")
	mrs = mrs.Remove("foo")
	checkMRs(t, mrs, "bar:1", ":-1")

	mrs = mrs.Remove("")
	checkMRs(t, mrs, "bar:1")
	mrs = mrs.Remove("")
	checkMRs(t, mrs, "bar:1")

	mrs = mrs.Remove("bar")
	td.CmpNil(t, mrs)
	mrs = mrs.Remove("bar")
	td.CmpNil(t, mrs)

	mrs = nil
	mrs = mrs.Add(newMR("DEFAULT", -1))
	mrs = mrs.Add(newMR("foo", 0))
	checkMRs(t, mrs, "foo:0", "DEFAULT:-1")
	mrs = mrs.Add(newMR("bar", 0))
	mrs = mrs.Add(newMR("bar", 1))
	checkMRs(t, mrs, "bar:1", "foo:0", "DEFAULT:-1")

	mrs = mrs.Remove("") // remove DEFAULT
	checkMRs(t, mrs, "bar:1", "foo:0")
	mrs = mrs.Remove("")
	checkMRs(t, mrs, "bar:1", "foo:0")

	mrs = mrs.Remove("bar")
	checkMRs(t, mrs, "foo:0")

	mrs = mrs.Remove("foo")
	td.CmpNil(t, mrs)
}

func TestMatchResponders_findMatchResponder(t *testing.T) {
	newReq := func() *http.Request {
		req, _ := http.NewRequest("GET", "/foo", ioutil.NopCloser(bytes.NewReader([]byte(`BODY`))))
		req.Header.Set("X-Foo", "bar")
		return req
	}

	assert := td.Assert(t).
		WithCmpHooks(
			func(a, b httpmock.MatchResponder) error {
				if a.Matcher().Name() != b.Matcher().Name() {
					return errors.New("name field mismatch")
				}
				if a.Matcher().FnPointer() != b.Matcher().FnPointer() {
					return errors.New("fn field mismatch")
				}
				if a.ResponderPointer() != b.ResponderPointer() {
					return errors.New("responder field mismatch")
				}
				return nil
			})

	var mrs httpmock.MatchResponders

	resp := httpmock.NewStringResponder(200, "OK")

	req := newReq()
	assert.Nil(mrs.FindMatchResponder(req))

	mrDefault := httpmock.NewMatchResponder(httpmock.Matcher{}, resp)
	mrs = mrs.Add(mrDefault)
	assert.Cmp(mrs.FindMatchResponder(req), &mrDefault)

	mrHeader1 := httpmock.NewMatchResponder(
		httpmock.NewMatcher("header-foo-zip", func(req *http.Request) bool {
			return req.Header.Get("X-Foo") == "zip"
		}),
		resp)
	mrs = mrs.Add(mrHeader1)
	assert.Cmp(mrs.FindMatchResponder(req), &mrDefault)

	mrHeader2 := httpmock.NewMatchResponder(
		httpmock.NewMatcher("header-foo-bar", func(req *http.Request) bool {
			return req.Header.Get("X-Foo") == "bar"
		}),
		resp)
	mrs = mrs.Add(mrHeader2)
	assert.Cmp(mrs.FindMatchResponder(req), &mrHeader2)

	mrs = mrs.Remove(mrHeader2.Matcher().Name()).
		Remove(mrDefault.Matcher().Name())
	assert.Nil(mrs.FindMatchResponder(req))

	mrBody1 := httpmock.NewMatchResponder(
		httpmock.NewMatcher("body-FOO", func(req *http.Request) bool {
			b, err := ioutil.ReadAll(req.Body)
			return err == nil && bytes.Equal(b, []byte("FOO"))
		}),
		resp)
	mrs = mrs.Add(mrBody1)

	req = newReq()
	assert.Nil(mrs.FindMatchResponder(req))

	mrBody2 := httpmock.NewMatchResponder(
		httpmock.NewMatcher("body-BODY", func(req *http.Request) bool {
			b, err := ioutil.ReadAll(req.Body)
			return err == nil && bytes.Equal(b, []byte("BODY"))
		}),
		resp)
	mrs = mrs.Add(mrBody2)

	req = newReq()
	assert.Cmp(mrs.FindMatchResponder(req), &mrBody2)

	// The request body should still be readable
	b, err := ioutil.ReadAll(req.Body)
	assert.CmpNoError(err)
	assert.String(b, "BODY")
}

func TestMatchRouteKey(t *testing.T) {
	td.Cmp(t, httpmock.NewMatchRouteKey(
		internal.RouteKey{
			Method: "GET",
			URL:    "/foo",
		},
		"").
		String(),
		"GET /foo")

	td.Cmp(t, httpmock.NewMatchRouteKey(
		internal.RouteKey{
			Method: "GET",
			URL:    "/foo",
		},
		"check-header").
		String(),
		"GET /foo <check-header>")
}

func TestBodyCopyOnRead(t *testing.T) {
	t.Run("non-nil body", func(t *testing.T) {
		body := ioutil.NopCloser(bytes.NewReader([]byte(`BODY`)))

		bc := httpmock.NewBodyCopyOnRead(body)

		bc.Rearm()
		td.Cmp(t, body, bc.Body(), "rearm didn't touch anything")

		var buf [4]byte
		n, err := bc.Read(buf[:])
		td.CmpNoError(t, err)
		td.Cmp(t, n, 4)
		td.CmpString(t, buf[:], "BODY")

		td.Cmp(t, body, td.Not(bc.Body()), "Original body has been copied internally")

		td.CmpNoError(t, bc.Body().Close()) // for coverage... :)

		n, err = bc.Read(buf[:])
		td.Cmp(t, err, io.EOF)
		td.Cmp(t, n, 0)

		bc.Rearm()

		n, err = bc.Read(buf[:])
		td.CmpNoError(t, err)
		td.Cmp(t, n, 4)
		td.CmpString(t, buf[:], "BODY")

		td.CmpNoError(t, bc.Close())
	})

	testCases := []struct {
		name string
		body io.ReadCloser
	}{
		{
			name: "nil body",
		},
		{
			name: "no body",
			body: http.NoBody,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bc := httpmock.NewBodyCopyOnRead(tc.body)

			bc.Rearm()
			td.Cmp(t, tc.body, bc.Body(), "rearm didn't touch anything")

			var buf [4]byte
			n, err := bc.Read(buf[:])
			td.Cmp(t, err, io.EOF)
			td.Cmp(t, n, 0)
			td.Cmp(t, bc.Body(), tc.body, "body is not altered")

			bc.Rearm()

			n, err = bc.Read(buf[:])
			td.Cmp(t, err, io.EOF)
			td.Cmp(t, n, 0)
			td.Cmp(t, bc.Body(), tc.body, "body is not altered")

			td.CmpNoError(t, bc.Close())
		})
	}
}

func TestExtractPackage(t *testing.T) {
	td.Cmp(t, httpmock.ExtractPackage("foo/bar/test.fn"), "foo/bar/test")
	td.Cmp(t, httpmock.ExtractPackage("foo/bar/test.X.fn"), "foo/bar/test")
	td.Cmp(t, httpmock.ExtractPackage("foo/bar/test.(*X).fn"), "foo/bar/test")
	td.Cmp(t, httpmock.ExtractPackage("foo/bar/test.(*X).fn.func1"), "foo/bar/test")
	td.Cmp(t, httpmock.ExtractPackage("weird"), "")
}

func TestIgnorePackages(t *testing.T) {
	ignorePackages := httpmock.GetIgnorePackages()

	td.Cmp(t, ignorePackages, td.Len(1))
	td.Cmp(t, ignorePackages, td.ContainsKey(td.HasSuffix("/httpmock")))

	httpmock.IgnoreMatcherHelper()
	td.Cmp(t, ignorePackages, td.Len(2), "current httpmock_test package added")
	td.Cmp(t, ignorePackages, td.ContainsKey(td.HasSuffix("/httpmock_test")))

	httpmock.IgnoreMatcherHelper(1)
	td.Cmp(t, ignorePackages, td.Len(3), "caller of TestIgnorePackages() → testing")
	td.Cmp(t, ignorePackages, td.ContainsKey("testing"))

	td.Cmp(t, httpmock.GetPackage(1000), "")
}

func TestCalledFrom(t *testing.T) {
	td.Cmp(t, httpmock.CalledFrom(0), td.Re(`^ @.*/httpmock_test\.TestCalledFrom\(\) .*/match_test.go:\d+\z`))

	td.Cmp(t, httpmock.CalledFrom(1000), "")
}
