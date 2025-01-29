package internal_test

import (
	"testing"

	"github.com/jarcoal/httpmock/internal"
)

func TestRouteKey(t *testing.T) {
	got, expected := internal.NoResponder.String(), "NO_RESPONDER"
	if got != expected {
		t.Errorf("got: %v, expected: %v", got, expected)
	}

	got, expected = internal.RouteKey{Method: "GET", URL: "/foo"}.String(), "GET /foo"
	if got != expected {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}
