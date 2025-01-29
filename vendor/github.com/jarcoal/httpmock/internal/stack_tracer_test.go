package internal_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock/internal"
)

func TestStackTracer(t *testing.T) {
	st := internal.StackTracer{}
	if st.Error() != "" {
		t.Errorf("Error() returned <%s> instead of <>", st.Error())
	}

	st = internal.StackTracer{
		Err: errors.New("foo"),
	}
	if st.Error() != "foo" {
		t.Errorf("Error() returned <%s> instead of <foo>", st.Error())
	}
}

func TestCheckStackTracer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://foo.bar/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// no error
	gotErr := internal.CheckStackTracer(req, nil)
	if gotErr != nil {
		t.Errorf(`CheckStackTracer(nil) should return nil, not %v`, gotErr)
	}

	// Classic error
	err = errors.New("error")
	gotErr = internal.CheckStackTracer(req, err)
	if err != gotErr {
		t.Errorf(`CheckStackTracer(err) should return %v, not %v`, err, gotErr)
	}

	// stackTracer without customFn
	origErr := errors.New("foo")
	errTracer := internal.StackTracer{
		Err: origErr,
	}
	gotErr = internal.CheckStackTracer(req, errTracer)
	if gotErr != origErr {
		t.Errorf(`Returned error mismatch, expected: %v, got: %v`, origErr, gotErr)
	}

	// stackTracer with nil error & without customFn
	errTracer = internal.StackTracer{}
	gotErr = internal.CheckStackTracer(req, errTracer)
	if gotErr != nil {
		t.Errorf(`Returned error mismatch, expected: nil, got: %v`, gotErr)
	}

	// stackTracer
	var mesg string
	errTracer = internal.StackTracer{
		Err: origErr,
		CustomFn: func(args ...interface{}) {
			mesg = args[0].(string)
		},
	}
	gotErr = internal.CheckStackTracer(req, errTracer)
	if !strings.HasPrefix(mesg, "foo\nCalled from ") || strings.HasSuffix(mesg, "\n") {
		t.Errorf(`mesg does not match "^foo\nCalled from .*[^\n]\z", it is "` + mesg + `"`)
	}
	if gotErr != origErr {
		t.Errorf(`Returned error mismatch, expected: %v, got: %v`, origErr, gotErr)
	}

	// stackTracer with nil error but customFn
	mesg = ""
	errTracer = internal.StackTracer{
		CustomFn: func(args ...interface{}) {
			mesg = args[0].(string)
		},
	}
	gotErr = internal.CheckStackTracer(req, errTracer)
	if !strings.HasPrefix(mesg, "GET http://foo.bar/\nCalled from ") || strings.HasSuffix(mesg, "\n") {
		t.Errorf(`mesg does not match "^foo\nCalled from .*[^\n]\z", it is "` + mesg + `"`)
	}
	if gotErr != nil {
		t.Errorf(`Returned error mismatch, expected: nil, got: %v`, gotErr)
	}
}
