package internal_test

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock/internal"
)

func TestSubmatches(t *testing.T) {
	req, err := http.NewRequest("GET", "/foo/bar", nil)
	if err != nil {
		t.Fatal(err)
	}

	var req2 *http.Request

	req2 = internal.SetSubmatches(req, nil)
	if req2 != req {
		t.Error("SetSubmatches(req, nil) should return the same request")
	}

	sm := internal.GetSubmatches(req2)
	if sm != nil {
		t.Errorf("GetSubmatches() should return nil")
	}

	req2 = internal.SetSubmatches(req, []string{})
	if req2 != req {
		t.Error("SetSubmatches(req, []string{}) should return the same request")
	}

	sm = internal.GetSubmatches(req2)
	if sm != nil {
		t.Errorf("GetSubmatches() should return nil")
	}

	req2 = internal.SetSubmatches(req, []string{"foo", "123", "-123", "12.3"})
	if req2 == req {
		t.Error("setSubmatches(req, []string{...}) should NOT return the same request")
	}

	sm = internal.GetSubmatches(req2)
	if len(sm) != 4 {
		t.Errorf("GetSubmatches() should return 4 items")
	}
}
