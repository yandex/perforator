/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package errhttp

import (
	"errors"
	"net/http"
	"testing"

	"github.com/containerd/errdefs"
)

func TestHTTPNilInput(t *testing.T) {
	if rc := ToHTTP(nil); rc != http.StatusInternalServerError {
		t.Fatalf("Expected %d error, got %d", http.StatusInternalServerError, rc)
	}
}

func TestHTTPRoundTrip(t *testing.T) {
	errShouldLeaveAlone := errors.New("unknown to package")

	for _, testcase := range []struct {
		input error
		cause error
		str   string
	}{
		{
			input: errdefs.ErrInvalidArgument,
			cause: errdefs.ErrInvalidArgument,
		},
		{
			input: errdefs.ErrNotFound,
			cause: errdefs.ErrNotFound,
		},
		{
			input: errdefs.ErrConflict,
			cause: errdefs.ErrConflict,
		},
		{
			input: errdefs.ErrNotModified,
			cause: errdefs.ErrNotModified,
		},
		{
			input: errdefs.ErrFailedPrecondition,
			cause: errdefs.ErrFailedPrecondition,
		},
		{
			input: errdefs.ErrUnauthenticated,
			cause: errdefs.ErrUnauthenticated,
		},
		{
			input: errdefs.ErrPermissionDenied,
			cause: errdefs.ErrPermissionDenied,
		},
		{
			input: errdefs.ErrResourceExhausted,
			cause: errdefs.ErrResourceExhausted,
		},
		{
			input: errdefs.ErrInternal,
			cause: errdefs.ErrInternal,
		},
		{
			input: errdefs.ErrNotImplemented,
			cause: errdefs.ErrNotImplemented,
		},
		{
			input: errdefs.ErrUnavailable,
			cause: errdefs.ErrUnavailable,
		},
		{
			input: errShouldLeaveAlone,
			cause: errdefs.ErrInternal,
		},
	} {
		t.Run(testcase.input.Error(), func(t *testing.T) {
			t.Logf("input: %v", testcase.input)
			httpErr := ToHTTP(testcase.input)
			t.Logf("http: %v", httpErr)
			ferr := ToNative(httpErr)
			t.Logf("recovered: %v", ferr)

			if !errors.Is(ferr, testcase.cause) {
				t.Fatalf("unexpected cause: !errors.Is(%v, %v)", ferr, testcase.cause)
			}

			expected := testcase.str
			if expected == "" {
				expected = testcase.cause.Error()
			}
			if ferr.Error() != expected {
				t.Fatalf("unexpected string: %q != %q", ferr.Error(), expected)
			}
		})
	}
}
