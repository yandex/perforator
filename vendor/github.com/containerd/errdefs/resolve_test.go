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

package errdefs

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestResolve(t *testing.T) {
	for i, tc := range []struct {
		err      error
		resolved error
	}{
		{nil, nil},
		{wrap(ErrUnknown), ErrUnknown},
		{wrap(ErrNotFound), ErrNotFound},
		{wrap(ErrInvalidArgument), ErrInvalidArgument},
		{wrap(ErrNotFound), ErrNotFound},
		{wrap(ErrAlreadyExists), ErrAlreadyExists},
		{wrap(ErrPermissionDenied), ErrPermissionDenied},
		{wrap(ErrResourceExhausted), ErrResourceExhausted},
		{wrap(ErrFailedPrecondition), ErrFailedPrecondition},
		{wrap(ErrConflict), ErrConflict},
		{wrap(ErrNotModified), ErrNotModified},
		{wrap(ErrAborted), ErrAborted},
		{wrap(ErrOutOfRange), ErrOutOfRange},
		{wrap(ErrNotImplemented), ErrNotImplemented},
		{wrap(ErrInternal), ErrInternal},
		{wrap(ErrUnavailable), ErrUnavailable},
		{wrap(ErrDataLoss), ErrDataLoss},
		{wrap(ErrUnauthenticated), ErrUnauthenticated},
		{wrap(context.DeadlineExceeded), context.DeadlineExceeded},
		{wrap(context.Canceled), context.Canceled},
		{errors.Join(errors.New("untyped"), wrap(ErrInvalidArgument)), ErrInvalidArgument},
		{errors.Join(ErrConflict, ErrNotFound), ErrConflict},
		{errors.New("untyped"), ErrUnknown},
		{errors.Join(wrap(ErrUnauthenticated), ErrNotModified), ErrUnauthenticated},
		{ErrDataLoss, ErrDataLoss},
		{errors.Join(ErrOutOfRange), ErrOutOfRange},
		{errors.Join(ErrNotImplemented, ErrInternal), ErrNotImplemented},
		{context.Canceled, context.Canceled},
		{testUnavailable{}, ErrUnavailable},
		{wrap(testUnavailable{}), ErrUnavailable},
		{errors.Join(testUnavailable{}, ErrPermissionDenied), ErrUnavailable},
		{errors.Join(errors.New("untyped join")), ErrUnknown},
		{errors.Join(errors.New("untyped1"), errors.New("untyped2")), ErrUnknown},
		{ErrNotFound.WithMessage("something else"), ErrNotFound},
		{wrap(ErrNotFound.WithMessage("something else")), ErrNotFound},
		{errors.Join(ErrNotFound.WithMessage("something else"), ErrPermissionDenied), ErrNotFound},
	} {
		name := fmt.Sprintf("%d-%s", i, errorString(tc.resolved))
		tc := tc
		t.Run(name, func(t *testing.T) {
			resolved := Resolve(tc.err)
			if resolved != tc.resolved {
				t.Errorf("Expected %s, got %s", tc.resolved, resolved)
			}
		})
	}
}

func wrap(err error) error {
	err = fmt.Errorf("wrapped error: %w", err)
	return fmt.Errorf("%w and also %w", err, ErrUnknown)
}

func errorString(err error) string {
	if err == nil {
		return "nil"
	}
	return err.Error()
}

type testUnavailable struct{}

func (testUnavailable) Error() string { return "" }
func (testUnavailable) Unavailable()  {}
