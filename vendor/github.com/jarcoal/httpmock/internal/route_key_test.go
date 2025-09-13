package internal_test

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock/internal"
)

func TestRouteKey(t *testing.T) {
	td.Cmp(t, internal.NoResponder.String(), "NO_RESPONDER")

	td.Cmp(t, internal.RouteKey{Method: "GET", URL: "/foo"}.String(), "GET /foo")
}
