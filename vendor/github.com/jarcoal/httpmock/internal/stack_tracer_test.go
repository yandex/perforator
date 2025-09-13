package internal_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock/internal"
)

func TestStackTracer(t *testing.T) {
	st := internal.StackTracer{}
	td.CmpEmpty(t, st.Error())

	st = internal.StackTracer{
		Err: errors.New("foo"),
	}
	td.Cmp(t, st.Error(), "foo")

	td.Cmp(t, st.Unwrap(), st.Err)
}

func TestCheckStackTracer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://foo.bar/", nil)
	td.Require(t).CmpNoError(err)

	// no error
	td.CmpNoError(t, internal.CheckStackTracer(req, nil))

	// Classic error
	err = errors.New("error")
	td.Cmp(t, internal.CheckStackTracer(req, err), err)

	// stackTracer without customFn
	origErr := errors.New("foo")
	errTracer := internal.StackTracer{
		Err: origErr,
	}
	td.Cmp(t, internal.CheckStackTracer(req, errTracer), origErr)

	// stackTracer with nil error & without customFn
	errTracer = internal.StackTracer{}
	td.CmpNoError(t, internal.CheckStackTracer(req, errTracer))

	// stackTracer
	var mesg string
	errTracer = internal.StackTracer{
		Err: origErr,
		CustomFn: func(args ...interface{}) {
			mesg = args[0].(string)
		},
	}
	gotErr := internal.CheckStackTracer(req, errTracer)
	td.Cmp(t, mesg, td.Re(`(?s)^foo\nCalled from .*[^\n]\z`))
	td.Cmp(t, gotErr, origErr)

	// stackTracer with nil error but customFn
	mesg = ""
	errTracer = internal.StackTracer{
		CustomFn: func(args ...interface{}) {
			mesg = args[0].(string)
		},
	}
	gotErr = internal.CheckStackTracer(req, errTracer)
	td.Cmp(t, mesg, td.Re(`(?s)^GET http://foo\.bar/\nCalled from .*[^\n]\z`))
	td.CmpNoError(t, gotErr)
}
