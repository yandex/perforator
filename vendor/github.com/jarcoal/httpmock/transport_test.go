package httpmock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil" //nolint: staticcheck
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/maxatome/go-testdeep/td"

	. "github.com/jarcoal/httpmock"
	"github.com/jarcoal/httpmock/internal"
)

const testURL = "http://www.example.com/"

func TestMockTransport(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/foo/bar"
	body := `["hello world"]` + "\n"

	RegisterResponder("GET", url, NewStringResponder(200, body))
	RegisterResponder("GET", `=~/xxx\z`, NewStringResponder(200, body))

	assert := td.Assert(t)

	// Read it as a simple string (ioutil.ReadAll of assertBody will
	// trigger io.EOF)
	assert.RunAssertRequire("simple", func(assert, require *td.T) {
		resp, err := http.Get(url)
		require.CmpNoError(err)
		assertBody(assert, resp, body)

		// the http client wraps our NoResponderFound error, so we just try and match on text
		_, err = http.Get(testURL)
		assert.HasSuffix(err, NoResponderFound.Error())

		// Use wrongly cased method, the error should warn us
		req, err := http.NewRequest("Get", url, nil)
		require.CmpNoError(err)

		c := http.Client{}
		_, err = c.Do(req)
		assert.HasSuffix(err, NoResponderFound.Error()+` for method "Get", but one matches method "GET"`)

		// Use POST instead of GET, the error should warn us
		req, err = http.NewRequest("POST", url, nil)
		require.CmpNoError(err)

		_, err = c.Do(req)
		assert.HasSuffix(err, NoResponderFound.Error()+` for method "POST", but one matches method "GET"`)

		// Same using a regexp responder
		req, err = http.NewRequest("POST", "http://pipo.com/xxx", nil)
		require.CmpNoError(err)

		_, err = c.Do(req)
		assert.HasSuffix(err, NoResponderFound.Error()+` for method "POST", but one matches method "GET"`)

		// Use a URL with squashable "/" in path
		_, err = http.Get("https://github.com////foo//bar")
		assert.HasSuffix(err, NoResponderFound.Error()+` for URL "https://github.com////foo//bar", but one matches URL "https://github.com/foo/bar"`)

		// Use a URL terminated by "/"
		_, err = http.Get("https://github.com/foo/bar/")
		assert.HasSuffix(err, NoResponderFound.Error()+` for URL "https://github.com/foo/bar/", but one matches URL "https://github.com/foo/bar"`)
	})

	// Do it again, but twice with json decoder (json Decode will not
	// reach EOF, but Close is called as the JSON response is complete)
	for i := 1; i <= 2; i++ {
		assert.RunAssertRequire(fmt.Sprintf("try #%d", i), func(assert, require *td.T) {
			resp, err := http.Get(url)
			require.CmpNoError(err)
			defer resp.Body.Close()

			var res []string
			err = json.NewDecoder(resp.Body).Decode(&res)
			require.CmpNoError(err)

			assert.Cmp(res, []string{"hello world"})
		})
	}
}

func TestRegisterMatcherResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	RegisterMatcherResponder("POST", "/foo",
		NewMatcher(
			"00-header-foo=bar",
			func(r *http.Request) bool { return r.Header.Get("Foo") == "bar" },
		),
		NewStringResponder(200, "header-foo"))

	RegisterMatcherResponder("POST", "/foo",
		NewMatcher(
			"01-body-BAR",
			func(r *http.Request) bool {
				b, err := ioutil.ReadAll(r.Body)
				return err == nil && bytes.Contains(b, []byte("BAR"))
			}),
		NewStringResponder(200, "body-BAR"))

	RegisterMatcherResponder("POST", "/foo",
		NewMatcher(
			"02-body-FOO",
			func(r *http.Request) bool {
				b, err := ioutil.ReadAll(r.Body)
				return err == nil && bytes.Contains(b, []byte("FOO"))
			}),
		NewStringResponder(200, "body-FOO"))

	RegisterMatcherResponder("POST", "/foo",
		BodyContainsString("xxx").
			Or(BodyContainsString("yyy")).
			WithName("03-body-xxx|yyy"),
		NewStringResponder(200, "body-xxx|yyy"))

	RegisterResponder("POST", "/foo", NewStringResponder(200, "default"))

	RegisterNoResponder(NewNotFoundResponder(nil))

	testCases := []struct {
		name         string
		body         string
		fooHeader    string
		expectedBody string
	}{
		{
			name:         "header",
			body:         "pipo",
			fooHeader:    "bar",
			expectedBody: "header-foo",
		},
		{
			name:         "header+body=header",
			body:         "BAR",
			fooHeader:    "bar",
			expectedBody: "header-foo",
		},
		{
			name:         "body BAR",
			body:         "BAR",
			fooHeader:    "xxx",
			expectedBody: "body-BAR",
		},
		{
			name:         "body FOO",
			body:         "FOO",
			expectedBody: "body-FOO",
		},
		{
			name:         "body xxx",
			body:         "...xxx...",
			expectedBody: "body-xxx|yyy",
		},
		{
			name:         "body yyy",
			body:         "...yyy...",
			expectedBody: "body-xxx|yyy",
		},
		{
			name:         "default",
			body:         "ANYTHING",
			fooHeader:    "zzz",
			expectedBody: "default",
		},
	}
	assert := td.Assert(t)
	for _, tc := range testCases {
		assert.RunAssertRequire(tc.name, func(assert, require *td.T) {
			req, err := http.NewRequest(
				"POST",
				"http://test.com/foo",
				strings.NewReader(tc.body),
			)
			require.CmpNoError(err)

			req.Header.Set("Content-Type", "text/plain")
			if tc.fooHeader != "" {
				req.Header.Set("Foo", tc.fooHeader)
			}

			resp, err := http.DefaultClient.Do(req)
			require.CmpNoError(err)

			assertBody(assert, resp, tc.expectedBody)
		})
	}

	// Remove the default responder
	RegisterResponder("POST", "/foo", nil)

	assert.Run("not found despite 3", func(assert *td.T) {
		_, err := http.Post(
			"http://test.com/foo",
			"text/plain",
			strings.NewReader("ANYTHING"),
		)
		assert.HasSuffix(err, `Responder not found for POST http://test.com/foo despite 4 matchers: ["00-header-foo=bar" "01-body-BAR" "02-body-FOO" "03-body-xxx|yyy"]`)
	})

	// Remove 3 matcher responders
	RegisterMatcherResponder("POST", "/foo", NewMatcher("01-body-BAR", nil), nil)
	RegisterMatcherResponder("POST", "/foo", NewMatcher("02-body-FOO", nil), nil)
	RegisterMatcherResponder("POST", "/foo", NewMatcher("03-body-xxx|yyy", nil), nil)

	assert.Run("not found despite 1", func(assert *td.T) {
		_, err := http.Post(
			"http://test.com/foo",
			"text/plain",
			strings.NewReader("ANYTHING"),
		)
		assert.HasSuffix(err, `Responder not found for POST http://test.com/foo despite matcher "00-header-foo=bar"`)
	})

	// Add a regexp responder without a Matcher: as the exact match
	// didn't succeed because of the "00-header-foo=bar" Matcher, the
	// following one must be tried ans also succeed
	RegisterResponder("POST", "=~^/foo", NewStringResponder(200, "regexp"))

	assert.RunAssertRequire("default regexp", func(assert, require *td.T) {
		resp, err := http.Post(
			"http://test.com/foo",
			"text/plain",
			strings.NewReader("ANYTHING"),
		)
		// The exact match responder "00-header-foo=bar" fails because of
		// its Matcher, so regexp responders have to be checked and ^/foo
		// has to match
		require.CmpNoError(err)
		assertBody(assert, resp, "regexp")
	})

	// Remove the previous regexp responder
	RegisterResponder("POST", "=~^/foo", nil)

	// Add a regexp Matcher responder that should match ZIP body
	RegisterMatcherResponder("POST", "=~^/foo",
		BodyContainsString("ZIP").WithName("10-body-ZIP"),
		NewStringResponder(200, "body-ZIP"))

	assert.RunAssertRequire("regexp matcher OK", func(assert, require *td.T) {
		resp, err := http.Post(
			"http://test.com/foo",
			"text/plain",
			strings.NewReader("ZIP"),
		)
		// The exact match responder "00-header-foo=bar" fails because of
		// its Matcher, so regexp responders have to be checked and ^/foo
		// + body ZIP has to match
		require.CmpNoError(err)
		assertBody(assert, resp, "body-ZIP")
	})

	assert.Run("regexp matcher no match", func(assert *td.T) {
		_, err := http.Post(
			"http://test.com/foo",
			"text/plain",
			strings.NewReader("ANYTHING"),
		)
		// The exact match responder "00-header-foo=bar" fails because of
		// its Matcher, so regexp responders have to be checked BUT none
		// match. In this case the returned error has to be the first
		// encountered, so the one corresponding to the exact match phase,
		// not the regexp one
		assert.HasSuffix(err, `Responder not found for POST http://test.com/foo despite matcher "00-header-foo=bar"`)
	})
}

// We should be able to find GET handlers when using an http.Request with a
// default (zero-value) .Method.
func TestMockTransportDefaultMethod(t *testing.T) {
	assert, require := td.AssertRequire(t)

	Activate()
	defer Deactivate()

	const urlString = "https://github.com/"
	url, err := url.Parse(urlString)
	require.CmpNoError(err)
	body := "hello world"

	RegisterResponder("GET", urlString, NewStringResponder(200, body))

	req := &http.Request{
		URL: url,
		// Note: Method unspecified (zero-value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	require.CmpNoError(err)

	assertBody(assert, resp, body)
}

func TestMockTransportReset(t *testing.T) {
	DeactivateAndReset()

	td.CmpZero(t, DefaultTransport.NumResponders(),
		"expected no responders at this point")
	td.Cmp(t, DefaultTransport.Responders(), []string{})

	r := NewStringResponder(200, "hey")

	RegisterResponder("GET", testURL, r)
	RegisterResponder("POST", testURL, r)
	RegisterResponder("PATCH", testURL, r)
	RegisterResponder("GET", "/pipo/bingo", r)

	RegisterResponder("GET", "=~/pipo/bingo", r)
	RegisterResponder("GET", "=~/bingo/pipo", r)

	td.Cmp(t, DefaultTransport.NumResponders(), 6, "expected one responder")
	td.Cmp(t, DefaultTransport.Responders(), []string{
		// Sorted by URL then method
		"GET /pipo/bingo",
		"GET " + testURL,
		"PATCH " + testURL,
		"POST " + testURL,
		// Regexp responders, in the same order they have been registered
		"GET =~/pipo/bingo",
		"GET =~/bingo/pipo",
	})

	Reset()

	td.CmpZero(t, DefaultTransport.NumResponders(),
		"expected no responders as they were just reset")
	td.Cmp(t, DefaultTransport.Responders(), []string{})
}

func TestMockTransportNoResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	Reset()

	_, err := http.Get(testURL)
	td.CmpError(t, err, "expected to receive a connection error due to lack of responders")

	RegisterNoResponder(NewStringResponder(200, "hello world"))

	resp, err := http.Get(testURL)
	if td.CmpNoError(t, err, "expected request to succeed") {
		assertBody(t, resp, "hello world")
	}

	// Using NewNotFoundResponder()
	RegisterNoResponder(NewNotFoundResponder(nil))
	_, err = http.Get(testURL)
	td.CmpHasSuffix(t, err, "Responder not found for GET http://www.example.com/")

	const url = "http://www.example.com/foo/bar"
	RegisterResponder("POST", url, NewStringResponder(200, "hello world"))

	// Help the user in case a Responder exists for another method
	_, err = http.Get(url)
	td.CmpHasSuffix(t, err, `Responder not found for GET `+url+`, but one matches method "POST"`)

	// Help the user in case a Responder exists for another path without final "/"
	_, err = http.Post(url+"/", "", nil)
	td.CmpHasSuffix(t, err, `Responder not found for POST `+url+`/, but one matches URL "`+url+`"`)

	// Help the user in case a Responder exists for another path without double "/"
	_, err = http.Post("http://www.example.com///foo//bar", "", nil)
	td.CmpHasSuffix(t, err, `Responder not found for POST http://www.example.com///foo//bar, but one matches URL "`+url+`"`)
}

func TestMockTransportQuerystringFallback(t *testing.T) {
	assert := td.Assert(t)

	Activate()
	defer DeactivateAndReset()

	// register the testURL responder
	RegisterResponder("GET", testURL, NewStringResponder(200, "hello world"))

	for _, suffix := range []string{"?", "?hello=world", "?hello=world#foo", "?hello=world&hello=all", "#foo"} {
		assert.RunAssertRequire(suffix, func(assert, require *td.T) {
			reqURL := testURL + suffix

			// make a request for the testURL with a querystring
			resp, err := http.Get(reqURL)
			require.CmpNoError(err)

			assertBody(assert, resp, "hello world")
		})
	}
}

func TestMockTransportPathOnlyFallback(t *testing.T) {
	// Just in case a panic occurs
	defer DeactivateAndReset()

	for _, test := range []struct {
		Responder string
		Paths     []string
	}{
		{
			// unsorted query string matches exactly
			Responder: "/hello/world?query=string&abc=zz#fragment",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
			},
		},
		{
			// sorted query string matches all cases
			Responder: "/hello/world?abc=zz&query=string#fragment",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?abc=zz&query=string#fragment",
			},
		},
		{
			// unsorted query string matches exactly
			Responder: "/hello/world?query=string&abc=zz",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz",
			},
		},
		{
			// sorted query string matches all cases
			Responder: "/hello/world?abc=zz&query=string",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world?abc=zz&query=string",
			},
		},
		{
			// unsorted query string matches exactly
			Responder: "/hello/world?query=string&query=string2&abc=zz",
			Paths: []string{
				testURL + "hello/world?query=string&query=string2&abc=zz",
			},
		},
		// sorted query string matches all cases
		{
			Responder: "/hello/world?abc=zz&query=string&query=string2",
			Paths: []string{
				testURL + "hello/world?query=string&query=string2&abc=zz",
				testURL + "hello/world?query=string2&query=string&abc=zz",
				testURL + "hello/world?abc=zz&query=string2&query=string",
			},
		},
		{
			Responder: "/hello/world?query",
			Paths: []string{
				testURL + "hello/world?query",
			},
		},
		{
			Responder: "/hello/world?query&abc",
			Paths: []string{
				testURL + "hello/world?query&abc",
				// testURL + "hello/world?abc&query" won't work as "=" is needed, see below
			},
		},
		{
			// In case the sorting does not matter for received params without
			// values, we must register params with "="
			Responder: "/hello/world?abc=&query=",
			Paths: []string{
				testURL + "hello/world?query&abc",
				testURL + "hello/world?abc&query",
			},
		},
		{
			Responder: "/hello/world#fragment",
			Paths: []string{
				testURL + "hello/world#fragment",
			},
		},
		{
			Responder: "/hello/world",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: "/hello%2fworl%64",
			Paths: []string{
				testURL + "hello%2fworl%64?query=string&abc=zz#fragment",
				testURL + "hello%2fworl%64?query=string&abc=zz",
				testURL + "hello%2fworl%64#fragment",
				testURL + "hello%2fworl%64",
			},
		},
		// Regexp cases
		{
			Responder: `=~^http://.*/hello/.*ld\z`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~^http://.*/hello/.*ld(\z|[?#])`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~^/hello/.*ld\z`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~^/hello/.*ld(\z|[?#])`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~abc=zz`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
			},
		},
	} {
		Activate()

		// register the responder
		RegisterResponder("GET", test.Responder, NewStringResponder(200, "hello world"))

		for _, reqURL := range test.Paths {
			t.Logf("%s: %s", test.Responder, reqURL)

			// make a request for the testURL with a querystring
			resp, err := http.Get(reqURL)
			if td.CmpNoError(t, err) {
				assertBody(t, resp, "hello world")
			}
		}

		DeactivateAndReset()
	}
}

type dummyTripper struct{}

func (d *dummyTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestMockTransportInitialTransport(t *testing.T) {
	DeactivateAndReset()

	tripper := &dummyTripper{}
	http.DefaultTransport = tripper

	Activate()

	td.CmpNot(t, http.DefaultTransport, td.Shallow(tripper),
		"expected http.DefaultTransport to be a mock transport")

	Deactivate()

	td.Cmp(t, http.DefaultTransport, td.Shallow(tripper),
		"expected http.DefaultTransport to be dummy")
}

func TestMockTransportNonDefault(t *testing.T) {
	assert, require := td.AssertRequire(t)

	// create a custom http client w/ custom Roundtripper
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}

	// activate mocks for the client
	ActivateNonDefault(client)
	defer DeactivateAndReset()

	body := "hello world!"

	RegisterResponder("GET", testURL, NewStringResponder(200, body))

	req, err := http.NewRequest("GET", testURL, nil)
	require.CmpNoError(err)

	resp, err := client.Do(req)
	require.CmpNoError(err)

	assertBody(assert, resp, body)
}

func TestMockTransportRespectsCancel(t *testing.T) {
	assert := td.Assert(t)

	Activate()
	defer DeactivateAndReset()

	const (
		cancelNone = iota
		cancelReq
		cancelCtx
	)

	cases := []struct {
		withCancel   int
		cancelNow    bool
		withPanic    bool
		expectedBody string
		expectedErr  error
	}{
		// No cancel specified at all. Falls back to normal behavior
		{cancelNone, false, false, "hello world", nil},

		// Cancel returns error
		{cancelReq, true, false, "", errors.New("request canceled")},

		// Cancel via context returns error
		{cancelCtx, true, false, "", errors.New("context canceled")},

		// Request can be cancelled but it is not cancelled.
		{cancelReq, false, false, "hello world", nil},

		// Request can be cancelled but it is not cancelled.
		{cancelCtx, false, false, "hello world", nil},

		// Panic in cancelled request is handled
		{cancelReq, false, true, "", errors.New(`panic in responder: got "oh no"`)},

		// Panic in cancelled request is handled
		{cancelCtx, false, true, "", errors.New(`panic in responder: got "oh no"`)},
	}

	for ic, c := range cases {
		assert.RunAssertRequire(fmt.Sprintf("case #%d", ic), func(assert, require *td.T) {
			Reset()
			if c.withPanic {
				RegisterResponder("GET", testURL, func(r *http.Request) (*http.Response, error) {
					time.Sleep(10 * time.Millisecond)
					panic("oh no")
				})
			} else {
				RegisterResponder("GET", testURL, func(r *http.Request) (*http.Response, error) {
					time.Sleep(10 * time.Millisecond)
					return NewStringResponse(http.StatusOK, "hello world"), nil
				})
			}

			req, err := http.NewRequest("GET", testURL, nil)
			require.CmpNoError(err)

			switch c.withCancel {
			case cancelReq:
				cancel := make(chan struct{}, 1)
				req.Cancel = cancel // nolint: staticcheck
				if c.cancelNow {
					cancel <- struct{}{}
				}
			case cancelCtx:
				ctx, cancel := context.WithCancel(req.Context())
				req = req.WithContext(ctx)
				if c.cancelNow {
					cancel()
				} else {
					defer cancel() // avoid ctx leak
				}
			}

			resp, err := http.DefaultClient.Do(req)

			if c.expectedErr != nil {
				// err is a *url.Error here, so with a Err field
				assert.Cmp(err, td.Smuggle("Err", td.String(c.expectedErr.Error())))
			} else {
				assert.CmpNoError(err)
			}

			if c.expectedBody != "" {
				assertBody(assert, resp, c.expectedBody)
			}
		})
	}
}

func TestMockTransportRespectsTimeout(t *testing.T) {
	timeout := time.Millisecond
	client := &http.Client{
		Timeout: timeout,
	}

	ActivateNonDefault(client)
	defer DeactivateAndReset()

	RegisterResponder(
		"GET", testURL,
		func(r *http.Request) (*http.Response, error) {
			time.Sleep(100 * timeout)
			return NewStringResponse(http.StatusOK, ""), nil
		},
	)

	_, err := client.Get(testURL)
	td.CmpError(t, err)
}

func TestMockTransportCallCountReset(t *testing.T) {
	assert, require := td.AssertRequire(t)

	Reset()
	Activate()
	defer Deactivate()

	const (
		url  = "https://github.com/path?b=1&a=2"
		url2 = "https://gitlab.com/"
	)

	RegisterResponder("GET", url, NewStringResponder(200, "body"))
	RegisterResponder("POST", "=~gitlab", NewStringResponder(200, "body"))
	RegisterMatcherResponder("POST", "=~gitlab",
		BodyContainsString("pipo").WithName("pipo-in-body"),
		NewStringResponder(200, "body"))

	_, err := http.Get(url)
	require.CmpNoError(err)

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode("{}") // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	require.CmpNoError(err)

	buff.Reset()
	json.NewEncoder(buff).Encode(`{"pipo":"bingo"}`) // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	require.CmpNoError(err)

	_, err = http.Get(url)
	require.CmpNoError(err)

	assert.Cmp(GetTotalCallCount(), 2+1+1)
	assert.Cmp(GetCallCountInfo(), map[string]int{
		"GET " + url: 2,
		// Regexp match generates 2 entries:
		"POST " + url2:  1, // the matched call
		"POST =~gitlab": 1, // the regexp responder
		// Regexp + matcher match also generates 2 entries:
		"POST " + url2 + " <pipo-in-body>": 1, // matched call
		"POST =~gitlab <pipo-in-body>":     1, // the regexp responder with matcher
	})

	Reset()

	assert.Zero(GetTotalCallCount())
	assert.Empty(GetCallCountInfo())
}

func TestMockTransportCallCountZero(t *testing.T) {
	assert, require := td.AssertRequire(t)

	Reset()
	Activate()
	defer Deactivate()

	const (
		url  = "https://github.com/path?b=1&a=2"
		url2 = "https://gitlab.com/"
	)

	RegisterResponder("GET", url, NewStringResponder(200, "body"))
	RegisterResponder("POST", "=~gitlab", NewStringResponder(200, "body"))
	RegisterMatcherResponder("POST", "=~gitlab",
		BodyContainsString("pipo").WithName("pipo-in-body"),
		NewStringResponder(200, "body"))

	_, err := http.Get(url)
	require.CmpNoError(err)

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode("{}") // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	require.CmpNoError(err)

	buff.Reset()
	json.NewEncoder(buff).Encode(`{"pipo":"bingo"}`) // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	require.CmpNoError(err)

	_, err = http.Get(url)
	require.CmpNoError(err)

	assert.Cmp(GetTotalCallCount(), 2+1+1)
	assert.Cmp(GetCallCountInfo(), map[string]int{
		"GET " + url: 2,
		// Regexp match generates 2 entries:
		"POST " + url2:  1, // the matched call
		"POST =~gitlab": 1, // the regexp responder
		// Regexp + matcher match also generates 2 entries:
		"POST " + url2 + " <pipo-in-body>": 1, // matched call
		"POST =~gitlab <pipo-in-body>":     1, // the regexp responder with matcher
	})

	ZeroCallCounters()

	assert.Zero(GetTotalCallCount())
	assert.Cmp(GetCallCountInfo(), map[string]int{
		"GET " + url: 0,
		// Regexp match generates 2 entries:
		"POST " + url2:  0, // the matched call
		"POST =~gitlab": 0, // the regexp responder
		// Regexp + matcher match also generates 2 entries:
		"POST " + url2 + " <pipo-in-body>": 0, // matched call
		"POST =~gitlab <pipo-in-body>":     0, // the regexp responder with matcher
	})

	// Unregister each responder
	RegisterResponder("GET", url, nil)
	RegisterResponder("POST", "=~gitlab", nil)
	RegisterMatcherResponder("POST", "=~gitlab", NewMatcher("pipo-in-body", nil), nil)

	assert.Cmp(GetCallCountInfo(), map[string]int{
		// these ones remain as they are not directly related to a
		// registered responder but a consequence of a regexp match
		"POST " + url2:                     0,
		"POST " + url2 + " <pipo-in-body>": 0,
	})
}

func TestRegisterResponderWithQuery(t *testing.T) {
	assert, require := td.AssertRequire(t)

	Reset()

	// Just in case a panic occurs
	defer DeactivateAndReset()

	// create a custom http client w/ custom Roundtripper
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}

	body := "hello world!"
	testURLPath := "http://acme.test/api"

	for _, test := range []struct {
		URL     string
		Queries []interface{}
		URLs    []string
	}{
		{
			Queries: []interface{}{
				map[string]string{"a": "1", "b": "2"},
				"a=1&b=2",
				"b=2&a=1",
				url.Values{"a": []string{"1"}, "b": []string{"2"}},
			},
			URLs: []string{
				"http://acme.test/api?a=1&b=2",
				"http://acme.test/api?b=2&a=1",
			},
		},
		{
			Queries: []interface{}{
				url.Values{
					"a": []string{"3", "2", "1"},
					"b": []string{"4", "2"},
					"c": []string{""}, // is the net/url way to record params without values
					// Test:
					//   u, _ := url.Parse("/hello/world?query")
					//   fmt.Printf("%d<%s>\n", len(u.Query()["query"]), u.Query()["query"][0])
					//   // prints "1<>"
				},
				"a=1&b=2&a=3&c&b=4&a=2",
				"b=2&a=1&c=&b=4&a=2&a=3",
				nil,
			},
			URLs: []string{
				testURLPath + "?a=1&b=2&a=3&c&b=4&a=2",
				testURLPath + "?a=1&b=2&a=3&c=&b=4&a=2",
				testURLPath + "?b=2&a=1&c=&b=4&a=2&a=3",
				testURLPath + "?b=2&a=1&c&b=4&a=2&a=3",
			},
		},
	} {
		for _, query := range test.Queries {
			ActivateNonDefault(client)
			RegisterResponderWithQuery("GET", testURLPath, query, NewStringResponder(200, body))

			for _, url := range test.URLs {
				assert.Logf("query=%v URL=%s", query, url)

				req, err := http.NewRequest("GET", url, nil)
				require.CmpNoError(err)

				resp, err := client.Do(req)
				require.CmpNoError(err)

				assertBody(assert, resp, body)
			}

			if info := GetCallCountInfo(); len(info) != 1 {
				t.Fatalf("%s: len(GetCallCountInfo()) should be 1 but contains %+v", testURLPath, info)
			}

			// Remove...
			RegisterResponderWithQuery("GET", testURLPath, query, nil)
			require.Len(GetCallCountInfo(), 0)

			for _, url := range test.URLs {
				t.Logf("query=%v URL=%s", query, url)

				req, err := http.NewRequest("GET", url, nil)
				require.CmpNoError(err)

				_, err = client.Do(req)
				assert.HasSuffix(err, "no responder found")
			}

			DeactivateAndReset()
		}
	}
}

func TestRegisterResponderWithQueryPanic(t *testing.T) {
	resp := NewStringResponder(200, "hello world!")

	for _, test := range []struct {
		Path        string
		Query       interface{}
		PanicPrefix string
	}{
		{
			Path:        "foobar",
			Query:       "%",
			PanicPrefix: "RegisterResponderWithQuery bad query string: ",
		},
		{
			Path:        "foobar",
			Query:       1234,
			PanicPrefix: "RegisterResponderWithQuery bad query type int. Only url.Values, map[string]string and string are allowed",
		},
		{
			Path:        `=~regexp.*\z`,
			Query:       "",
			PanicPrefix: `path begins with "=~", RegisterResponder should be used instead of RegisterResponderWithQuery`,
		},
	} {
		td.CmpPanic(t,
			func() { RegisterResponderWithQuery("GET", test.Path, test.Query, resp) },
			td.HasPrefix(test.PanicPrefix),
			`RegisterResponderWithQuery + query=%v`, test.Query)
	}
}

func TestRegisterRegexpResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	rx := regexp.MustCompile("ex.mple")

	RegisterRegexpResponder("POST", rx, NewStringResponder(200, "first"))
	// Overwrite responder
	RegisterRegexpResponder("POST", rx, NewStringResponder(200, "second"))

	resp, err := http.Post(testURL, "text/plain", strings.NewReader("PIPO"))
	td.Require(t).CmpNoError(err)
	assertBody(t, resp, "second")

	RegisterRegexpMatcherResponder("POST", rx,
		BodyContainsString("PIPO").WithName("01-body-PIPO"),
		NewStringResponder(200, "matcher-PIPO"))

	RegisterRegexpMatcherResponder("POST", rx,
		BodyContainsString("BINGO").WithName("02-body-BINGO"),
		NewStringResponder(200, "matcher-BINGO"))

	resp, err = http.Post(testURL, "text/plain", strings.NewReader("PIPO"))
	td.Require(t).CmpNoError(err)
	assertBody(t, resp, "matcher-PIPO")

	resp, err = http.Post(testURL, "text/plain", strings.NewReader("BINGO"))
	td.Require(t).CmpNoError(err)
	assertBody(t, resp, "matcher-BINGO")

	// Remove 01-body-PIPO matcher
	RegisterRegexpMatcherResponder("POST", rx, NewMatcher("01-body-PIPO", nil), nil)

	resp, err = http.Post(testURL, "text/plain", strings.NewReader("PIPO"))
	td.Require(t).CmpNoError(err)
	assertBody(t, resp, "second")

	resp, err = http.Post(testURL, "text/plain", strings.NewReader("BINGO"))
	td.Require(t).CmpNoError(err)
	assertBody(t, resp, "matcher-BINGO")

	// Remove 02-body-BINGO matcher
	RegisterRegexpMatcherResponder("POST", rx, NewMatcher("02-body-BINGO", nil), nil)

	resp, err = http.Post(testURL, "text/plain", strings.NewReader("BINGO"))
	td.Require(t).CmpNoError(err)
	assertBody(t, resp, "second")
}

func TestSubmatches(t *testing.T) {
	assert, require := td.AssertRequire(t)

	req, err := http.NewRequest("GET", "/foo/bar", nil)
	require.CmpNoError(err)

	req2 := internal.SetSubmatches(req, []string{"foo", "123", "-123", "12.3"})

	assert.Run("GetSubmatch", func(assert *td.T) {
		_, err := GetSubmatch(req, 1)
		assert.Cmp(err, ErrSubmatchNotFound)

		_, err = GetSubmatch(req2, 5)
		assert.Cmp(err, ErrSubmatchNotFound)

		s, err := GetSubmatch(req2, 1)
		assert.CmpNoError(err)
		assert.Cmp(s, "foo")

		s, err = GetSubmatch(req2, 4)
		assert.CmpNoError(err)
		assert.Cmp(s, "12.3")

		s = MustGetSubmatch(req2, 4)
		assert.Cmp(s, "12.3")
	})

	assert.Run("GetSubmatchAsInt", func(assert *td.T) {
		_, err := GetSubmatchAsInt(req, 1)
		assert.Cmp(err, ErrSubmatchNotFound)

		_, err = GetSubmatchAsInt(req2, 4) // not an int
		assert.CmpError(err)
		assert.Not(err, ErrSubmatchNotFound)

		i, err := GetSubmatchAsInt(req2, 3)
		assert.CmpNoError(err)
		assert.CmpLax(i, -123)

		i = MustGetSubmatchAsInt(req2, 3)
		assert.CmpLax(i, -123)
	})

	assert.Run("GetSubmatchAsUint", func(assert *td.T) {
		_, err := GetSubmatchAsUint(req, 1)
		assert.Cmp(err, ErrSubmatchNotFound)

		_, err = GetSubmatchAsUint(req2, 3) // not a uint
		assert.CmpError(err)
		assert.Not(err, ErrSubmatchNotFound)

		u, err := GetSubmatchAsUint(req2, 2)
		assert.CmpNoError(err)
		assert.CmpLax(u, 123)

		u = MustGetSubmatchAsUint(req2, 2)
		assert.CmpLax(u, 123)
	})

	assert.Run("GetSubmatchAsFloat", func(assert *td.T) {
		_, err := GetSubmatchAsFloat(req, 1)
		assert.Cmp(err, ErrSubmatchNotFound)

		_, err = GetSubmatchAsFloat(req2, 1) // not a float
		assert.CmpError(err)
		assert.Not(err, ErrSubmatchNotFound)

		f, err := GetSubmatchAsFloat(req2, 4)
		assert.CmpNoError(err)
		assert.Cmp(f, 12.3)

		f = MustGetSubmatchAsFloat(req2, 4)
		assert.Cmp(f, 12.3)
	})

	assert.Run("GetSubmatch* panics", func(assert *td.T) {
		for _, test := range []struct {
			Name        string
			Fn          func()
			PanicPrefix string
		}{
			{
				Name:        "GetSubmatch & n < 1",
				Fn:          func() { GetSubmatch(req, 0) }, // nolint: errcheck
				PanicPrefix: "getting submatches starts at 1, not 0",
			},
			{
				Name:        "MustGetSubmatch",
				Fn:          func() { MustGetSubmatch(req, 1) },
				PanicPrefix: "GetSubmatch failed: " + ErrSubmatchNotFound.Error(),
			},
			{
				Name:        "MustGetSubmatchAsInt",
				Fn:          func() { MustGetSubmatchAsInt(req2, 4) }, // not an int
				PanicPrefix: "GetSubmatchAsInt failed: ",
			},
			{
				Name:        "MustGetSubmatchAsUint",
				Fn:          func() { MustGetSubmatchAsUint(req2, 3) }, // not a uint
				PanicPrefix: "GetSubmatchAsUint failed: ",
			},
			{
				Name:        "GetSubmatchAsFloat",
				Fn:          func() { MustGetSubmatchAsFloat(req2, 1) }, // not a float
				PanicPrefix: "GetSubmatchAsFloat failed: ",
			},
		} {
			assert.CmpPanic(test.Fn, td.HasPrefix(test.PanicPrefix), test.Name)
		}
	})

	assert.RunAssertRequire("Full test", func(assert, require *td.T) {
		Activate()
		defer DeactivateAndReset()

		var (
			id       uint64
			delta    float64
			deltaStr string
			inc      int64
		)
		RegisterResponder("GET", `=~^/id/(\d+)\?delta=(\d+(?:\.\d*)?)&inc=(-?\d+)\z`,
			func(req *http.Request) (*http.Response, error) {
				id = MustGetSubmatchAsUint(req, 1)
				delta = MustGetSubmatchAsFloat(req, 2)
				deltaStr = MustGetSubmatch(req, 2)
				inc = MustGetSubmatchAsInt(req, 3)

				return NewStringResponse(http.StatusOK, "OK"), nil
			})

		resp, err := http.Get("http://example.tld/id/123?delta=1.2&inc=-5")
		require.CmpNoError(err)
		assertBody(assert, resp, "OK")

		// Check submatches
		assert.CmpLax(id, 123, "MustGetSubmatchAsUint")
		assert.Cmp(delta, 1.2, "MustGetSubmatchAsFloat")
		assert.Cmp(deltaStr, "1.2", "MustGetSubmatch")
		assert.CmpLax(inc, -5, "MustGetSubmatchAsInt")
	})
}

func TestCheckStackTracer(t *testing.T) {
	assert, require := td.AssertRequire(t)

	// Full test using Trace() Responder
	Activate()
	defer Deactivate()

	const url = "https://foo.bar/"
	var mesg string
	RegisterResponder("GET", url,
		NewStringResponder(200, "{}").
			Trace(func(args ...interface{}) { mesg = args[0].(string) }))

	resp, err := http.Get(url)
	require.CmpNoError(err)

	assertBody(assert, resp, "{}")

	// Check that first frame is the net/http.Get() call
	assert.HasPrefix(mesg, "GET https://foo.bar/\nCalled from net/http.Get()\n    at ")
	assert.Not(mesg, td.HasSuffix("\n"))
}

func TestCheckMethod(t *testing.T) {
	mt := NewMockTransport()

	const expected = `You probably want to use method "GET" instead of "get"? If not and so want to disable this check, set MockTransport.DontCheckMethod field to true`

	td.CmpPanic(t,
		func() { mt.RegisterResponder("get", "/pipo", NewStringResponder(200, "")) },
		expected)

	td.CmpPanic(t,
		func() { mt.RegisterRegexpResponder("get", regexp.MustCompile("."), NewStringResponder(200, "")) },
		expected)

	td.CmpPanic(t,
		func() { mt.RegisterResponderWithQuery("get", "/pipo", url.Values(nil), NewStringResponder(200, "")) },
		expected)

	//
	// No longer panics
	mt.DontCheckMethod = true
	td.CmpNotPanic(t,
		func() { mt.RegisterResponder("get", "/pipo", NewStringResponder(200, "")) })

	td.CmpNotPanic(t,
		func() { mt.RegisterRegexpResponder("get", regexp.MustCompile("."), NewStringResponder(200, "")) })

	td.CmpNotPanic(t,
		func() { mt.RegisterResponderWithQuery("get", "/pipo", url.Values(nil), NewStringResponder(200, "")) })
}
