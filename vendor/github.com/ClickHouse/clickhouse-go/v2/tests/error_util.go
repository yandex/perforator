package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertIsTimeoutError ensures that the error provided is a timeout error.
// It recursively unwraps the provided error and ensures the core error
// implements the Timeout() method and it returns true.
// context deadline error, os deadline error and poll deadline error each
// implement this and return true.
func AssertIsTimeoutError(t *testing.T, err error) {
	assert.True(t, isDeadlineExceededError(err), "error is not a timeout error: %#v", err)
}

type timeout interface {
	Timeout() bool
}

func isDeadlineExceededError(err error) bool {
	nerr, ok := unwrap(err).(timeout)
	if !ok {
		return false
	}

	return nerr.Timeout()
}

// unwrap recursively unwraps the error until it gets the core error
func unwrap(err error) error {
	if uerr := errors.Unwrap(err); uerr != nil {
		return unwrap(uerr)
	}
	return err
}
