package httpmock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/jarcoal/httpmock"
	"github.com/jarcoal/httpmock/internal"
)

const testURL = "http://www.example.com/"

func TestMockTransport(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/"
	body := `["hello world"]` + "\n"

	RegisterResponder("GET", url, NewStringResponder(200, body))

	// Read it as a simple string (ioutil.ReadAll of assertBody will
	// trigger io.EOF)
	func() {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}

		if !assertBody(t, resp, body) {
			t.FailNow()
		}

		// the http client wraps our NoResponderFound error, so we just try and match on text
		if _, err := http.Get(testURL); !strings.Contains(err.Error(), NoResponderFound.Error()) {
			t.Fatal(err)
		}
	}()

	// Do it again, but twice with json decoder (json Decode will not
	// reach EOF, but Close is called as the JSON response is complete)
	for i := 0; i < 2; i++ {
		func() {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			var res []string
			err = json.NewDecoder(resp.Body).Decode(&res)
			if err != nil {
				t.Fatal(err)
			}

			if len(res) != 1 || res[0] != "hello world" {
				t.Fatalf(`%v read instead of ["hello world"]`, res)
			}
		}()
	}
}

// We should be able to find GET handlers when using an http.Request with a
// default (zero-value) .Method.
func TestMockTransportDefaultMethod(t *testing.T) {
	Activate()
	defer Deactivate()

	const urlString = "https://github.com/"
	url, err := url.Parse(urlString)
	if err != nil {
		t.Fatal(err)
	}
	body := "hello world"

	RegisterResponder("GET", urlString, NewStringResponder(200, body))

	req := &http.Request{
		URL: url,
		// Note: Method unspecified (zero-value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assertBody(t, resp, body)
}

func TestMockTransportReset(t *testing.T) {
	DeactivateAndReset()

	if DefaultTransport.NumResponders() > 0 {
		t.Fatal("expected no responders at this point")
	}

	RegisterResponder("GET", testURL, nil)

	if DefaultTransport.NumResponders() != 1 {
		t.Fatal("expected one responder")
	}

	Reset()

	if DefaultTransport.NumResponders() > 0 {
		t.Fatal("expected no responders as they were just reset")
	}
}

func TestMockTransportNoResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	Reset()

	if _, err := http.Get(testURL); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	RegisterNoResponder(NewStringResponder(200, "hello world"))

	resp, err := http.Get(testURL)
	if err != nil {
		t.Fatal("expected request to succeed")
	}

	assertBody(t, resp, "hello world")
}

func TestMockTransportQuerystringFallback(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	// register the testURL responder
	RegisterResponder("GET", testURL, NewStringResponder(200, "hello world"))

	for _, suffix := range []string{"?", "?hello=world", "?hello=world#foo", "?hello=world&hello=all", "#foo"} {
		reqURL := testURL + suffix
		t.Log(reqURL)

		// make a request for the testURL with a querystring
		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("expected request %s to succeed", reqURL)
		}

		assertBody(t, resp, "hello world")
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
				// testURL + "hello/world?abc&query" won' work as "=" is needed, see below
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
			if err != nil {
				t.Errorf("%s: expected request %s to succeed", test.Responder, reqURL)
				continue
			}

			assertBody(t, resp, "hello world")
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

	if http.DefaultTransport == tripper {
		t.Fatal("expected http.DefaultTransport to be a mock transport")
	}

	Deactivate()

	if http.DefaultTransport != tripper {
		t.Fatal("expected http.DefaultTransport to be dummy")
	}
}

func TestMockTransportNonDefault(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assertBody(t, resp, body)
}

func TestMockTransportRespectsCancel(t *testing.T) {
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

	for _, c := range cases {
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
		if err != nil {
			t.Fatal(err)
		}

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

		// If we expect an error but none was returned, it's fatal for this test...
		if err == nil && c.expectedErr != nil {
			t.Fatal("Error should not be nil")
		}

		if err != nil {
			got := err.(*url.Error)
			// Do not use reflect.DeepEqual as go 1.13 includes stack frames
			// into errors issued by errors.New()
			if c.expectedErr == nil || got.Err.Error() != c.expectedErr.Error() {
				t.Errorf("Expected error: %v, got: %v", c.expectedErr, got.Err)
			}
		}

		if c.expectedBody != "" {
			assertBody(t, resp, c.expectedBody)
		}
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
	if err == nil {
		t.Fail()
	}
}

func TestMockTransportCallCountReset(t *testing.T) {
	Reset()
	Activate()
	defer Deactivate()

	const (
		url  = "https://github.com/path?b=1&a=2"
		url2 = "https://gitlab.com/"
	)

	RegisterResponder("GET", url, NewStringResponder(200, "body"))
	RegisterResponder("POST", "=~gitlab", NewStringResponder(200, "body"))

	_, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode("{}") // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	if err != nil {
		t.Fatal(err)
	}

	_, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	totalCallCount := GetTotalCallCount()
	if totalCallCount != 3 {
		t.Fatalf("did not track the total count of calls correctly. expected it to be 3, but it was %v", totalCallCount)
	}

	info := GetCallCountInfo()
	expectedInfo := map[string]int{
		"GET " + url: 2,
		// Regexp match generates 2 entries:
		"POST " + url2:  1, // the matched call
		"POST =~gitlab": 1, // the regexp responder
	}

	if !reflect.DeepEqual(info, expectedInfo) {
		t.Fatalf("did not correctly track the call count info. expected it to be \n %+v\n but it was \n %+v", expectedInfo, info)
	}

	Reset()

	afterResetTotalCallCount := GetTotalCallCount()
	if afterResetTotalCallCount != 0 {
		t.Fatalf("did not reset the total count of calls correctly. expected it to be 0 after reset, but it was %v", afterResetTotalCallCount)
	}

	info = GetCallCountInfo()
	if !reflect.DeepEqual(info, map[string]int{}) {
		t.Fatalf("did not correctly reset the call count info. expected it to be \n {}\n but it was \n %+v", info)
	}
}

func TestMockTransportCallCountZero(t *testing.T) {
	Reset()
	Activate()
	defer Deactivate()

	const (
		url  = "https://github.com/path?b=1&a=2"
		url2 = "https://gitlab.com/"
	)

	RegisterResponder("GET", url, NewStringResponder(200, "body"))
	RegisterResponder("POST", "=~gitlab", NewStringResponder(200, "body"))

	_, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode("{}") // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	if err != nil {
		t.Fatal(err)
	}

	_, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	totalCallCount := GetTotalCallCount()
	if totalCallCount != 3 {
		t.Fatalf("did not track the total count of calls correctly. expected it to be 3, but it was %v", totalCallCount)
	}

	info := GetCallCountInfo()
	expectedInfo := map[string]int{
		"GET " + url: 2,
		// Regexp match generates 2 entries:
		"POST " + url2:  1, // the matched call
		"POST =~gitlab": 1, // the regexp responder
	}

	if !reflect.DeepEqual(info, expectedInfo) {
		t.Fatalf("did not correctly track the call count info. expected it to be \n %+v\n but it was \n %+v", expectedInfo, info)
	}

	ZeroCallCounters()

	afterResetTotalCallCount := GetTotalCallCount()
	if afterResetTotalCallCount != 0 {
		t.Fatalf("did not reset the total count of calls correctly. expected it to be 0 after reset, but it was %v", afterResetTotalCallCount)
	}

	info = GetCallCountInfo()
	expectedInfo = map[string]int{
		"GET " + url: 0,
		// Regexp match generates 2 entries:
		"POST " + url2:  0, // the matched call
		"POST =~gitlab": 0, // the regexp responder
	}
	if !reflect.DeepEqual(info, expectedInfo) {
		t.Fatalf("did not correctly reset the call count info. expected it to be \n %+v\n but it was \n %+v", expectedInfo, info)
	}
}

func TestRegisterResponderWithQuery(t *testing.T) {
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
				t.Logf("query=%v URL=%s", query, url)

				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					t.Fatal(err)
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}

				assertBody(t, resp, body)
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
		panicked, panicStr := catchPanic(func() {
			RegisterResponderWithQuery("GET", test.Path, test.Query, resp)
		})

		if !panicked {
			t.Errorf("RegisterResponderWithQuery + query=%v did not panic", test.Query)
			continue
		}

		if !strings.HasPrefix(panicStr, test.PanicPrefix) {
			t.Fatalf(`RegisterResponderWithQuery + query=%v panic="%v" expected prefix="%v"`,
				test.Query, panicStr, test.PanicPrefix)
		}
	}
}

func TestRegisterRegexpResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	rx := regexp.MustCompile("ex.mple")

	RegisterRegexpResponder("GET", rx, NewStringResponder(200, "first"))
	// Overwrite responder
	RegisterRegexpResponder("GET", rx, NewStringResponder(200, "second"))

	resp, err := http.Get(testURL)
	if err != nil {
		t.Fatalf("expected request %s to succeed", testURL)
	}

	assertBody(t, resp, "second")
}

func TestSubmatches(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo/bar", nil)
	if err != nil {
		t.Fatal(err)
	}

	req2 := internal.SetSubmatches(req, []string{"foo", "123", "-123", "12.3"})

	t.Run("GetSubmatch", func(t *testing.T) {
		_, err := GetSubmatch(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatch(req2, 5)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch #5 should not be found in req2: %v", err)
		}

		s, err := GetSubmatch(req2, 1)
		if err != nil {
			t.Errorf("GetSubmatch(req2, 1) failed: %v", err)
		}
		if s != "foo" {
			t.Errorf("GetSubmatch(req2, 1) failed, got: %v, expected: foo", s)
		}

		s, err = GetSubmatch(req2, 4)
		if err != nil {
			t.Errorf("GetSubmatch(req2, 4) failed: %v", err)
		}
		if s != "12.3" {
			t.Errorf("GetSubmatch(req2, 4) failed, got: %v, expected: 12.3", s)
		}

		s = MustGetSubmatch(req2, 4)
		if s != "12.3" {
			t.Errorf("GetSubmatch(req2, 4) failed, got: %v, expected: 12.3", s)
		}
	})

	t.Run("GetSubmatchAsInt", func(t *testing.T) {
		_, err := GetSubmatchAsInt(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatchAsInt(req2, 4) // not an int
		if err == nil || err == ErrSubmatchNotFound {
			t.Errorf("Submatch should not be an int64: %v", err)
		}

		i, err := GetSubmatchAsInt(req2, 3)
		if err != nil {
			t.Errorf("GetSubmatchAsInt(req2, 3) failed: %v", err)
		}
		if i != -123 {
			t.Errorf("GetSubmatchAsInt(req2, 3) failed, got: %d, expected: -123", i)
		}

		i = MustGetSubmatchAsInt(req2, 3)
		if i != -123 {
			t.Errorf("MustGetSubmatchAsInt(req2, 3) failed, got: %d, expected: -123", i)
		}
	})

	t.Run("GetSubmatchAsUint", func(t *testing.T) {
		_, err := GetSubmatchAsUint(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatchAsUint(req2, 3) // not a uint
		if err == nil || err == ErrSubmatchNotFound {
			t.Errorf("Submatch should not be an uint64: %v", err)
		}

		u, err := GetSubmatchAsUint(req2, 2)
		if err != nil {
			t.Errorf("GetSubmatchAsUint(req2, 2) failed: %v", err)
		}
		if u != 123 {
			t.Errorf("GetSubmatchAsUint(req2, 2) failed, got: %d, expected: 123", u)
		}

		u = MustGetSubmatchAsUint(req2, 2)
		if u != 123 {
			t.Errorf("MustGetSubmatchAsUint(req2, 2) failed, got: %d, expected: 123", u)
		}
	})

	t.Run("GetSubmatchAsFloat", func(t *testing.T) {
		_, err := GetSubmatchAsFloat(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatchAsFloat(req2, 1) // not a float
		if err == nil || err == ErrSubmatchNotFound {
			t.Errorf("Submatch should not be an float64: %v", err)
		}

		f, err := GetSubmatchAsFloat(req2, 4)
		if err != nil {
			t.Errorf("GetSubmatchAsFloat(req2, 4) failed: %v", err)
		}
		if f != 12.3 {
			t.Errorf("GetSubmatchAsFloat(req2, 4) failed, got: %f, expected: 12.3", f)
		}

		f = MustGetSubmatchAsFloat(req2, 4)
		if f != 12.3 {
			t.Errorf("MustGetSubmatchAsFloat(req2, 4) failed, got: %f, expected: 12.3", f)
		}
	})

	t.Run("GetSubmatch* panics", func(t *testing.T) {
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
			var (
				didntPanic bool
				panicVal   interface{}
			)
			func() {
				defer func() { panicVal = recover() }()
				test.Fn()
				didntPanic = true
			}()

			if didntPanic {
				t.Errorf("%s did not panic", test.Name)
			}

			panicStr, ok := panicVal.(string)
			if !ok || !strings.HasPrefix(panicStr, test.PanicPrefix) {
				t.Errorf(`%s panic="%v" expected prefix="%v"`, test.Name, panicVal, test.PanicPrefix)
			}
		}
	})

	t.Run("Full test", func(t *testing.T) {
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
		if err != nil {
			t.Fatal(err)
		}
		assertBody(t, resp, "OK")

		// Check submatches
		if id != 123 {
			t.Errorf("seems MustGetSubmatchAsUint failed, got: %d, expected: 123", id)
		}
		if delta != 1.2 {
			t.Errorf("seems MustGetSubmatchAsFloat failed, got: %f, expected: 1.2", delta)
		}
		if deltaStr != "1.2" {
			t.Errorf("seems MustGetSubmatch failed, got: %v, expected: 1.2", deltaStr)
		}
		if inc != -5 {
			t.Errorf("seems MustGetSubmatchAsInt failed, got: %d, expected: 123", inc)
		}
	})
}

func TestCheckStackTracer(t *testing.T) {
	// Full test using Trace() Responder
	Activate()
	defer Deactivate()

	const url = "https://foo.bar/"
	var mesg string
	RegisterResponder("GET", url,
		NewStringResponder(200, "{}").
			Trace(func(args ...interface{}) { mesg = args[0].(string) }))

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	if !assertBody(t, resp, "{}") {
		t.FailNow()
	}

	// Check that first frame is the net/http.Get() call
	if !strings.HasPrefix(mesg, "GET https://foo.bar/\nCalled from net/http.Get()\n    at ") ||
		strings.HasSuffix(mesg, "\n") {
		t.Errorf("Bad mesg: <%v>", mesg)
	}
}
