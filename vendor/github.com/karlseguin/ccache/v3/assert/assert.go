// A wrapper around *testing.T. I hate the if a != b { t.ErrorF(....) } pattern.
// Packages should prefer using the tests package (which exposes all of
// these functions). The only reason to use this package directly is if
// the tests package depends on your package (and thus you have a cyclical
// dependency)
package assert

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

// a == b
func Equal[T comparable](t *testing.T, actual T, expected T) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected '%v' to equal '%v'", actual, expected)
		t.FailNow()
	}
}

// Two lists are equal (same length & same values in the same order)
func List[T comparable](t *testing.T, actuals []T, expecteds []T) {
	t.Helper()
	Equal(t, len(actuals), len(expecteds))

	for i, actual := range actuals {
		Equal(t, actual, expecteds[i])
	}
}

// needle not in []haystack
func DoesNotContain[T comparable](t *testing.T, haystack []T, needle T) {
	t.Helper()
	for _, v := range haystack {
		if v == needle {
			t.Errorf("expected '%v' to not be in '%v'", needle, haystack)
			t.FailNow()
		}
	}
}

// A value is nil
func Nil(t *testing.T, actual interface{}) {
	t.Helper()
	if actual != nil && !reflect.ValueOf(actual).IsNil() {
		t.Errorf("expected %v to be nil", actual)
		t.FailNow()
	}
}

// A value is not nil
func NotNil(t *testing.T, actual interface{}) {
	t.Helper()
	if actual == nil {
		t.Errorf("expected %v to be not nil", actual)
		t.FailNow()
	}
}

// A value is true
func True(t *testing.T, actual bool) {
	t.Helper()
	if !actual {
		t.Error("expected true, got false")
		t.FailNow()
	}
}

// A value is false
func False(t *testing.T, actual bool) {
	t.Helper()
	if actual {
		t.Error("expected false, got true")
		t.FailNow()
	}
}

// The string contains the given value
func StringContains(t *testing.T, actual string, expected string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Errorf("expected %s to contain %s", actual, expected)
		t.FailNow()
	}
}

func Error(t *testing.T, actual error, expected error) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected '%s' to be '%s'", actual, expected)
		t.FailNow()
	}
}

func Nowish(t *testing.T, actual time.Time) {
	t.Helper()
	diff := math.Abs(time.Now().UTC().Sub(actual).Seconds())
	if diff > 1 {
		t.Errorf("expected '%s' to be nowish", actual)
		t.FailNow()
	}
}
