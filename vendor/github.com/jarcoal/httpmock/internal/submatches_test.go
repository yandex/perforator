package internal_test

import (
	"net/http"
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock/internal"
)

func TestSubmatches(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo/bar", nil)
	td.Require(t).CmpNoError(err)

	var req2 *http.Request

	req2 = internal.SetSubmatches(req, nil)
	td.CmpShallow(t, req2, req)
	td.CmpNil(t, internal.GetSubmatches(req2))

	req2 = internal.SetSubmatches(req, []string{})
	td.Cmp(t, req2, td.Shallow(req))
	td.CmpNil(t, internal.GetSubmatches(req2))

	req2 = internal.SetSubmatches(req, []string{"foo", "123", "-123", "12.3"})
	td.CmpNot(t, req2, td.Shallow(req))
	td.CmpLen(t, internal.GetSubmatches(req2), 4)
}
