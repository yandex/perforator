// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

package assertions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/shoenig/test/interfaces"
	"github.com/shoenig/test/internal/constraints"
	"github.com/shoenig/test/wait"
)

const depth = 4

func Caller() string {
	_, file, line, ok := runtime.Caller(depth)
	if ok {
		file = filepath.Base(file)
		return fmt.Sprintf("%s:%d: ", file, line)
	}
	return "[???]"
}

// diff creates a diff of a and b using cmp.Diff if possible, falling back to printing
// the Go string values of both types (e.g. contains unexported fields).
func diff[A, B any](a A, b B, opts cmp.Options) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = fmt.Sprintf("↪ Assertion | comparison ↷\na: %#v\nb: %#v\n", a, b)
		}
	}()
	s = "↪ Assertion | differential ↷\n" + cmp.Diff(a, b, opts)
	return
}

// equal compares a and b using cmp.Equal if possible, falling back to reflect.DeepEqual
// (e.g. contains unexported fields).
func equal[A, B any](a A, b B, opts cmp.Options) (result bool) {
	defer func() {
		if r := recover(); r != nil {
			result = reflect.DeepEqual(a, b)
		}
	}()
	result = cmp.Equal(a, b, opts)
	return
}

func contains[C comparable](slice []C, item C) bool {
	found := false
	for i := 0; i < len(slice); i++ {
		if slice[i] == item {
			found = true
			break
		}
	}
	return found
}

func containsFunc[A, B any](slice []A, item B, eq func(a A, b B) bool) bool {
	found := false
	for i := 0; i < len(slice); i++ {
		if eq(slice[i], item) {
			found = true
			break
		}
	}
	return found
}

func isNil(a any) bool {
	// comparable check only works for simple types
	if a == nil {
		return true
	}

	// check for non-nil nil types
	value := reflect.ValueOf(a)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func Nil(a any) (s string) {
	if !isNil(a) {
		s = "expected to be nil; is not nil\n"
	}
	return
}

func NotNil(a any) (s string) {
	if isNil(a) {
		s = "expected to not be nil; is nil\n"
	}
	return
}

func True(condition bool) (s string) {
	if !condition {
		s = "expected condition to be true; is false\n"
	}
	return
}

func False(condition bool) (s string) {
	if condition {
		s = "expected condition to be false; is true\n"
	}
	return
}

func Unreachable() (s string) {
	s = "expected not to execute this code path\n"
	return
}

func Error(err error) (s string) {
	if err == nil {
		s = "expected non-nil error; is nil\n"
	}
	return
}

func EqError(err error, msg string) (s string) {
	if err == nil {
		s = "expected error; got nil\n"
		return
	}
	e := err.Error()
	if e != msg {
		s = "expected matching error strings\n"
		s += bullet("msg: %q\n", msg)
		s += bullet("err: %q\n", e)
	}
	return
}

func ErrorIs(err error, target error) (s string) {
	if err == nil {
		s = "expected error; got nil\n"
		return
	}
	if !errors.Is(err, target) {
		s = "expected errors.Is match\n"
		s += bullet(" error: %v\n", err)
		s += bullet("target: %v\n", target)
	}
	return
}

func NoError(err error) (s string) {
	if err != nil {
		s = "expected nil error\n"
		s += bullet("error: %v\n", err)
	}
	return
}

func ErrorContains(err error, sub string) (s string) {
	if err == nil {
		s = "expected non-nil error\n"
		return
	}
	actual := err.Error()
	if !strings.Contains(actual, sub) {
		s = "expected error to contain substring\n"
		s += bullet("substring: %s\n", sub)
		s += bullet("      err: %s\n", actual)
	}
	return
}

func Eq[A any](exp, val A, opts ...cmp.Option) (s string) {
	if !equal(exp, val, opts) {
		s = "expected equality via cmp.Equal function\n"
		s += diff(exp, val, opts)
	}
	return
}

func NotEq[A any](exp, val A, opts ...cmp.Option) (s string) {
	if equal(exp, val, opts) {
		s = "expected inequality via cmp.Equal function\n"
	}
	return
}

func EqOp[C comparable](exp, val C) (s string) {
	if exp != val {
		s = "expected equality via ==\n"
		s += diff(exp, val, nil)
	}
	return
}

func EqFunc[A any](exp, val A, eq func(a, b A) bool) (s string) {
	if !eq(exp, val) {
		s = "expected equality via 'eq' function\n"
		s += diff(exp, val, nil)
	}
	return
}

func NotEqOp[C comparable](exp, val C) (s string) {
	if exp == val {
		s = "expected inequality via !=\n"
	}
	return
}

func NotEqFunc[A any](exp, val A, eq func(a, b A) bool) (s string) {
	if eq(exp, val) {
		s = "expected inequality via 'eq' function\n"
	}
	return
}

func EqJSON(exp, val string) (s string) {
	var expA, expB any

	if err := json.Unmarshal([]byte(exp), &expA); err != nil {
		s = fmt.Sprintf("failed to unmarshal first argument as json: %v\n", err)
		return
	}

	if err := json.Unmarshal([]byte(val), &expB); err != nil {
		s = fmt.Sprintf("failed to unmarshal second argument as json: %v\n", err)
		return
	}

	if !reflect.DeepEqual(expA, expB) {
		jsonA, _ := json.Marshal(expA)
		jsonB, _ := json.Marshal(expB)
		s = "expected equality via json marshalling\n"
		s += diff(string(jsonA), string(jsonB), nil)
		return
	}

	return
}

func ValidJSON(input string) (s string) {
	return validJSON([]byte(input))
}

func ValidJSONBytes(input []byte) (s string) {
	return validJSON(input)
}

func validJSON(input []byte) (s string) {
	if !json.Valid([]byte(input)) {
		return "expected input to be valid json\n"
	}
	return
}

func EqSliceFunc[A, B any](exp []B, val []A, eq func(a A, b B) bool) (s string) {
	lenA, lenB := len(exp), len(val)

	if lenA != lenB {
		s = "expected slices of same length\n"
		s += bullet("len(exp): %d\n", lenA)
		s += bullet("len(val): %d\n", lenB)
		s += diff(exp, val, nil)
		return
	}

	miss := false
	for i := 0; i < lenA; i++ {
		if !eq(val[i], exp[i]) {
			miss = true
			break
		}
	}

	if miss {
		s = "expected slice equality via 'eq' function\n"
		s += diff(exp, val, nil)
		return
	}

	return
}

func Equal[E interfaces.EqualFunc[E]](exp, val E) (s string) {
	if !val.Equal(exp) {
		s = "expected equality via .Equal method\n"
		s += diff(exp, val, nil)
	}
	return
}

func NotEqual[E interfaces.EqualFunc[E]](exp, val E) (s string) {
	if val.Equal(exp) {
		s = "expected inequality via .Equal method\n"
	}
	return
}

func SliceEqual[E interfaces.EqualFunc[E]](exp, val []E) (s string) {
	lenA, lenB := len(exp), len(val)

	if lenA != lenB {
		s = "expected slices of same length\n"
		s += bullet("len(exp): %d\n", lenA)
		s += bullet("len(val): %d\n", lenB)
		s += diff(exp, val, nil)
		return
	}

	for i := 0; i < lenA; i++ {
		if !exp[i].Equal(val[i]) {
			s += "expected slice equality via .Equal method\n"
			s += diff(exp[i], val[i], nil)
			return
		}
	}
	return
}

func Lesser[L interfaces.LessFunc[L]](exp, val L) (s string) {
	if !val.Less(exp) {
		s = "expected val to be less via .Less method\n"
		s += diff(exp, val, nil)
	}
	return
}

func SliceEmpty[A any](slice []A) (s string) {
	if len(slice) != 0 {
		s = "expected slice to be empty\n"
		s += bullet("len(slice): %d\n", len(slice))
	}
	return
}

func SliceNotEmpty[A any](slice []A) (s string) {
	if len(slice) == 0 {
		s = "expected slice to not be empty\n"
		s += bullet("len(slice): %d\n", len(slice))
	}
	return
}

func SliceLen[A any](n int, slice []A) (s string) {
	if l := len(slice); l != n {
		s = "expected slice to be different length\n"
		s += bullet("len(slice): %d, expected: %d\n", l, n)
	}
	return
}

func SliceContainsOp[C comparable](slice []C, item C) (s string) {
	if !contains(slice, item) {
		s = "expected slice to contain missing item via == operator\n"
		s += bullet("slice is missing %#v\n", item)
	}
	return
}

func SliceContainsFunc[A, B any](slice []A, item B, eq func(a A, b B) bool) (s string) {
	if !containsFunc(slice, item, eq) {
		s = "expected slice to contain missing item via 'eq' function\n"
		s += bullet("slice is missing %#v\n", item)
	}
	return
}

func SliceContainsEqual[E interfaces.EqualFunc[E]](slice []E, item E) (s string) {
	if !containsFunc(slice, item, E.Equal) {
		s = "expected slice to contain missing item via .Equal method\n"
		s += bullet("slice is missing %#v\n", item)
	}
	return
}

func SliceContains[A any](slice []A, item A, opts ...cmp.Option) (s string) {
	for _, i := range slice {
		if cmp.Equal(i, item, opts...) {
			return
		}
	}
	s = "expected slice to contain missing item via cmp.Equal method\n"
	s += bullet("slice is missing %#v\n", item)
	return
}

func SliceNotContains[A any](slice []A, item A, opts ...cmp.Option) (s string) {
	for _, i := range slice {
		if cmp.Equal(i, item, opts...) {
			s = "expected slice to not contain item but it does\n"
			s += bullet("unwanted item %#v\n", item)
			return
		}
	}
	return
}

func SliceNotContainsFunc[A, B any](slice []A, item B, eq func(a A, b B) bool) (s string) {
	if containsFunc(slice, item, eq) {
		s = "expected slice to not contain item but it does\n"
		s += bullet("unwanted item %#v\n", item)
	}
	return
}

func SliceContainsAll[A any](slice, items []A, opts ...cmp.Option) (s string) {
	if len(slice) != len(items) {
		s = "expected slice and items to contain same number of elements\n"
		s += bullet("len(slice): %d\n", len(slice))
		s += bullet("len(items): %d\n", len(items))
		return s
	}
	return SliceContainsSubset(slice, items, opts...)
}

func SliceContainsSubset[A any](slice, items []A, opts ...cmp.Option) (s string) {
OUTER:
	for _, target := range items {
		var item A
		for _, item = range slice {
			if cmp.Equal(target, item, opts...) {
				continue OUTER
			}
		}
		s = "expected slice to contain missing item\n"
		s += bullet("slice is missing %#v\n", item)
		return
	}
	return
}

func Positive[N interfaces.Number](value N) (s string) {
	if !(value > 0) {
		s = "expected positive value\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func NonPositive[N interfaces.Number](value N) (s string) {
	if !(value <= 0) {
		s = "expected non-positive value\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func Negative[N interfaces.Number](value N) (s string) {
	if value > 0 {
		s = "expected negative value\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func NonNegative[N interfaces.Number](value N) (s string) {
	if !(value >= 0) {
		s = "expected non-negative value\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func Zero[N interfaces.Number](value N) (s string) {
	if value != 0 {
		s = "expected value of 0\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func NonZero[N interfaces.Number](value N) (s string) {
	if value == 0 {
		s = "expected non-zero value\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func One[N interfaces.Number](value N) (s string) {
	if value != 1 {
		s = "expected value of 1\n"
		s += bullet("value: %v\n", value)
	}
	return
}

func Less[O constraints.Ordered](exp, val O) (s string) {
	if !(val < exp) {
		s = fmt.Sprintf("expected %v < %v\n", val, exp)
	}
	return
}

func LessEq[O constraints.Ordered](exp, val O) (s string) {
	if !(val <= exp) {
		s = fmt.Sprintf("expected %v ≤ %v\n", val, exp)
	}
	return
}

func Greater[O constraints.Ordered](exp, val O) (s string) {
	if !(val > exp) {
		s = fmt.Sprintf("expected %v > %v\n", val, exp)
	}
	return
}

func GreaterEq[O constraints.Ordered](exp, val O) (s string) {
	if !(val >= exp) {
		s = fmt.Sprintf("expected %v ≥ %v\n", val, exp)
	}
	return
}

func Between[O constraints.Ordered](lower, val, upper O) (s string) {
	if val < lower || val > upper {
		s = fmt.Sprintf("expected val in range (%v ≤ val ≤ %v)\n", lower, upper)
		s += bullet("val: %v\n", val)
		return
	}
	return
}

func BetweenExclusive[O constraints.Ordered](lower, val, upper O) (s string) {
	if val <= lower || val >= upper {
		s = fmt.Sprintf("expected val in range (%v < val < %v)\n", lower, upper)
		s += bullet("val: %v\n", val)
		return
	}
	return
}

func Min[A any, C interfaces.MinFunc[A]](expect A, collection C, opts ...cmp.Option) (s string) {
	min := collection.Min()
	if !equal(expect, min, opts) {
		s = "expected a different value for min\n"
		s += diff(expect, min, opts)
	}
	return
}

func Max[A any, C interfaces.MaxFunc[A]](expect A, collection C, opts ...cmp.Option) (s string) {
	max := collection.Max()
	if !equal(expect, max, opts) {
		s = "expected a different value for max\n"
		s += diff(expect, max, opts)
	}
	return
}

func Ascending[O constraints.Ordered](slice []O) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		if slice[i] > slice[i+1] {
			s = fmt.Sprintf("expected slice[%d] <= slice[%d]\n", i, i+1)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func AscendingFunc[A any](slice []A, less func(a, b A) bool) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		if !less(slice[i], slice[i+1]) {
			s = fmt.Sprintf("expected less(slice[%d], slice[%d])\n", i, i+1)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func AscendingCmp[A any](slice []A, compare func(a, b A) int) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		cmp := compare(slice[i], slice[i+1])
		if cmp > 0 {
			s = fmt.Sprintf("expected compare(slice[%d], slice[%d]) <= 0\n", i, i+1)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func AscendingLess[L interfaces.LessFunc[L]](slice []L) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		if !slice[i].Less(slice[i+1]) {
			s = fmt.Sprintf("expected slice[%d].Less(slice[%d])\n", i, i+1)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func Descending[O constraints.Ordered](slice []O) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		if slice[i] < slice[i+1] {
			s = fmt.Sprintf("expected slice[%d] >= slice[%d]\n", i, i+1)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func DescendingFunc[A any](slice []A, less func(a, b A) bool) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		if !less(slice[i+1], slice[i]) {
			s = fmt.Sprintf("expected less(slice[%d], slice[%d])\n", i+1, i)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func DescendingCmp[A any](slice []A, compare func(a, b A) int) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		cmp := compare(slice[i], slice[i+1])
		if cmp < 0 {
			s = fmt.Sprintf("expected compare(slice[%d], slice[%d]) >= 0\n", i, i+1)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func DescendingLess[L interfaces.LessFunc[L]](slice []L) (s string) {
	for i := 0; i < len(slice)-1; i++ {
		if !(slice[i+1].Less(slice[i])) {
			s = fmt.Sprintf("expected slice[%d].Less(slice[%d])\n", i+1, i)
			s += bullet("slice[%d]: %v\n", i, slice[i])
			s += bullet("slice[%d]: %v\n", i+1, slice[i+1])
			return
		}
	}
	return
}

func InDelta[N interfaces.Number](a, b, delta N) (s string) {
	var zero N

	if !interfaces.Numeric(delta) {
		s = fmt.Sprintf("delta must be numeric; got %v\n", delta)
		return
	}

	if delta <= zero {
		s = fmt.Sprintf("delta must be positive; got %v\n", delta)
		return
	}

	if !interfaces.Numeric(a) {
		s = fmt.Sprintf("first argument must be numeric; got %v\n", a)
		return
	}

	if !interfaces.Numeric(b) {
		s = fmt.Sprintf("second argument must be numeric; got %v\n", b)
		return
	}

	difference := a - b
	if difference < -delta || difference > delta {
		s = fmt.Sprintf("%v and %v not within %v\n", a, b, delta)
		return
	}

	return
}

func InDeltaSlice[N interfaces.Number](a, b []N, delta N) (s string) {
	if len(a) != len(b) {
		s = "expected slices of same length\n"
		s += bullet("len(slice a): %d\n", len(a))
		s += bullet("len(slice b): %d\n", len(b))
		return
	}

	for i := 0; i < len(a); i++ {
		if s = InDelta(a[i], b[i], delta); s != "" {
			return
		}
	}
	return
}

func MapEq[M1, M2 interfaces.Map[K, V], K comparable, V any](exp M1, val M2, opts cmp.Options) (s string) {
	lenA, lenB := len(exp), len(val)

	if lenA != lenB {
		s = "expected maps of same length\n"
		s += bullet("len(exp): %d\n", lenA)
		s += bullet("len(val): %d\n", lenB)
		return
	}

	for key, valA := range exp {
		valB, exists := val[key]
		if !exists {
			s = "expected maps of same keys\n"
			s += diff(exp, val, opts)
			return
		}

		if !cmp.Equal(valA, valB, opts) {
			s = "expected maps of same values via cmp.Equal function\n"
			s += diff(exp, val, opts)
			return
		}
	}
	return
}

func MapEqFunc[M1, M2 interfaces.Map[K, V], K comparable, V any](exp M1, val M2, eq func(V, V) bool) (s string) {
	lenA, lenB := len(exp), len(val)

	if lenA != lenB {
		s = "expected maps of same length\n"
		s += bullet("len(exp): %d\n", lenA)
		s += bullet("len(val): %d\n", lenB)
		return
	}

	for key, valA := range exp {
		valB, exists := val[key]
		if !exists {
			s = "expected maps of same keys\n"
			s += diff(exp, val, nil)
			return
		}

		if !eq(valA, valB) {
			s = "expected maps of same values via 'eq' function\n"
			s += diff(exp, val, nil)
			return
		}
	}
	return
}

func MapEqual[M interfaces.MapEqualFunc[K, V], K comparable, V interfaces.EqualFunc[V]](exp, val M) (s string) {
	lenA, lenB := len(exp), len(val)

	if lenA != lenB {
		s = "expected maps of same length\n"
		s += bullet("len(exp): %d\n", lenA)
		s += bullet("len(val): %d\n", lenB)
		return
	}

	for key, valA := range exp {
		valB, exists := val[key]
		if !exists {
			s = "expected maps of same keys\n"
			s += diff(exp, val, nil)
			return
		}

		if !(valB).Equal(valA) {
			s = "expected maps of same values via .Equal method\n"
			s += diff(exp, val, nil)
			return
		}
	}

	return
}

func MapLen[M ~map[K]V, K comparable, V any](n int, m M) (s string) {
	if l := len(m); l != n {
		s = "expected map to be different length\n"
		s += bullet("len(map): %d, expected: %d\n", l, n)
	}
	return
}

func MapEmpty[M ~map[K]V, K comparable, V any](m M) (s string) {
	if l := len(m); l > 0 {
		s = "expected map to be empty\n"
		s += bullet("len(map): %d\n", l)
	}
	return
}

func MapNotEmpty[M ~map[K]V, K comparable, V any](m M) (s string) {
	if l := len(m); l == 0 {
		s = "expected map to not be empty\n"
		s += bullet("len(map): %d\n", l)
	}
	return
}

func MapContainsKey[M ~map[K]V, K comparable, V any](m M, key K) (s string) {
	if _, exists := m[key]; !exists {
		s = "expected map to contain key\n"
		s += bullet("key: %v\n", key)
	}
	return
}

func MapNotContainsKey[M ~map[K]V, K comparable, V any](m M, key K) (s string) {
	if _, exists := m[key]; exists {
		s = "expected map to not contain key\n"
		s += bullet("key: %v\n", key)
	}
	return
}

func MapContainsKeys[M ~map[K]V, K comparable, V any](m M, keys []K) (s string) {
	var missing []K
	for _, key := range keys {
		if _, exists := m[key]; !exists {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		s = "expected map to contain keys\n"
		for _, key := range missing {
			s += bullet("key: %v\n", key)
		}
	}
	return
}

func MapNotContainsKeys[M ~map[K]V, K comparable, V any](m M, keys []K) (s string) {
	var unwanted []K
	for _, key := range keys {
		if _, exists := m[key]; exists {
			unwanted = append(unwanted, key)
		}
	}
	if len(unwanted) > 0 {
		s = "expected map to not contain keys\n"
		for _, key := range unwanted {
			s += bullet("key: %v\n", key)
		}
	}
	return
}

func mapContains[M ~map[K]V, K comparable, V any](m M, values []V, eq func(V, V) bool) (s string) {
	var missing []V
	for _, wanted := range values {
		found := false
		for _, v := range m {
			if eq(wanted, v) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, wanted)
		}
	}

	if len(missing) > 0 {
		s = "expected map to contain values\n"
		for _, val := range missing {
			s += bullet("val: %v\n", val)
		}
	}
	return
}

func mapNotContains[M ~map[K]V, K comparable, V any](m M, values []V, eq func(V, V) bool) (s string) {
	var unexpected []V
	for _, target := range values {
		found := false
		for _, v := range m {
			if eq(target, v) {
				found = true
				break
			}
		}
		if found {
			unexpected = append(unexpected, target)
		}
	}
	if len(unexpected) > 0 {
		s = "expected map to not contain values\n"
		for _, val := range unexpected {
			s += bullet("val: %v\n", val)
		}
	}
	return
}

func MapContainsValues[M ~map[K]V, K comparable, V any](m M, vals []V, opts cmp.Options) (s string) {
	return mapContains(m, vals, func(a, b V) bool {
		return equal(a, b, opts)
	})
}

func MapNotContainsValues[M ~map[K]V, K comparable, V any](m M, vals []V, opts cmp.Options) (s string) {
	return mapNotContains(m, vals, func(a, b V) bool {
		return equal(a, b, opts)
	})
}

func MapContainsValuesFunc[M ~map[K]V, K comparable, V any](m M, vals []V, eq func(V, V) bool) (s string) {
	return mapContains(m, vals, eq)
}

func MapNotContainsValuesFunc[M ~map[K]V, K comparable, V any](m M, vals []V, eq func(V, V) bool) (s string) {
	return mapNotContains(m, vals, eq)
}

func MapContainsValuesEqual[M ~map[K]V, K comparable, V interfaces.EqualFunc[V]](m M, vals []V) (s string) {
	return mapContains(m, vals, func(a, b V) bool {
		return a.Equal(b)
	})
}

func MapNotContainsValuesEqual[M ~map[K]V, K comparable, V interfaces.EqualFunc[V]](m M, vals []V) (s string) {
	return mapNotContains(m, vals, func(a, b V) bool {
		return a.Equal(b)
	})
}

func FileExistsFS(system fs.FS, file string) (s string) {
	info, err := fs.Stat(system, file)
	if errors.Is(err, fs.ErrNotExist) {
		s = "expected file to exist\n"
		s += bullet(" name: %s\n", file)
		s += bullet("error: %s\n", err)
		return
	}

	// other errors - file probably exists but cannot be read
	if info.IsDir() {
		s = "expected file but is a directory\n"
		s += bullet("name: %s\n", file)
		return
	}
	return
}

func FileNotExistsFS(system fs.FS, file string) (s string) {
	_, err := fs.Stat(system, file)
	if err == nil {
		s = "expected file to not exist\n"
		s += bullet("name: %s\n", file)
		return
	}
	if !errors.Is(err, fs.ErrNotExist) {
		s = "expected not existing file but got different error\n"
		s += bullet("error: %s\n", err)
		return
	}
	return
}

func DirExistsFS(system fs.FS, directory string) (s string) {
	info, err := fs.Stat(system, directory)
	if os.IsNotExist(err) {
		s = "expected directory to exist\n"
		s += bullet(" name: %s\n", directory)
		s += bullet("error: %s\n", err)
		return
	}
	// other errors - directory probably exists but cannot be read
	if !info.IsDir() {
		s = "expected directory but is a file\n"
		s += bullet("name: %s\n", directory)
		return
	}
	return
}

func DirNotExistsFS(system fs.FS, directory string) (s string) {
	_, err := fs.Stat(system, directory)
	if !errors.Is(err, fs.ErrNotExist) {
		s = "expected directory to not exist\n"
		s += bullet("name: %s\n", directory)
		return
	}
	return
}

func FileModeFS(system fs.FS, path string, permissions fs.FileMode) (s string) {
	info, err := fs.Stat(system, path)
	if err != nil {
		s = "expected to stat path\n"
		s += bullet(" name: %s\n", path)
		s += bullet("error: %s\n", err)
		return
	}

	mode := info.Mode()
	if permissions != mode {
		s = "expected different file permissions\n"
		s += bullet("name: %s\n", path)
		s += bullet(" exp: %s\n", permissions)
		s += bullet(" got: %s\n", mode)
	}
	return
}

func FileContainsFS(system fs.FS, file, content string) (s string) {
	b, err := fs.ReadFile(system, file)
	if err != nil {
		s = "expected to read file\n"
		s += bullet(" name: %s\n", file)
		s += bullet("error: %s\n", err)
		return
	}
	actual := string(b)
	if !strings.Contains(string(b), content) {
		s = "expected file contents\n"
		s += bullet("  name: %s\n", file)
		s += bullet("wanted: %s\n", content)
		s += bullet("actual: %s\n", actual)
		return
	}
	return
}

func FilePathValid(path string) (s string) {
	if !fs.ValidPath(path) {
		s = "expected valid file path\n"
	}
	return
}

func Close(c io.Closer) (s string) {
	err := c.Close()
	if err != nil {
		s = "calling Close failed\n"
		s += bullet("error: %v\n", err)
	}
	return
}

func StrEqFold(exp, val string) (s string) {
	if !strings.EqualFold(exp, val) {
		s = "expected strings to be equal ignoring case\n"
		s += bullet("exp: %s\n", exp)
		s += bullet("val: %s\n", val)
	}
	return
}

func StrNotEqFold(exp, val string) (s string) {
	if strings.EqualFold(exp, val) {
		s = "expected strings to not be equal ignoring case; but they are\n"
		s += bullet("exp: %s\n", exp)
		s += bullet("val: %s\n", val)
	}
	return
}

func StrContains(str, sub string) (s string) {
	if !strings.Contains(str, sub) {
		s = "expected string to contain substring; it does not\n"
		s += bullet("substring: %s\n", sub)
		s += bullet("   string: %s\n", str)
	}
	return
}

func StrContainsFold(str, sub string) (s string) {
	upperS := strings.ToUpper(str)
	upperSub := strings.ToUpper(sub)
	return StrContains(upperS, upperSub)
}

func StrNotContains(str, sub string) (s string) {
	if strings.Contains(str, sub) {
		s = "expected string to not contain substring; but it does\n"
		s += bullet("substring: %s\n", sub)
		s += bullet("   string: %s\n", str)
	}
	return
}

func StrNotContainsFold(str, sub string) (s string) {
	upperS := strings.ToUpper(str)
	upperSub := strings.ToUpper(sub)
	return StrNotContains(upperS, upperSub)
}

func StrContainsAny(str, chars string) (s string) {
	if !strings.ContainsAny(str, chars) {
		s = "expected string to contain one or more code points\n"
		s += bullet("code-points: %s\n", chars)
		s += bullet("     string: %s\n", str)
	}
	return
}

func StrNotContainsAny(str, chars string) (s string) {
	if strings.ContainsAny(str, chars) {
		s = "expected string to not contain code points; but it does\n"
		s += bullet("code-points: %s\n", chars)
		s += bullet("     string: %s\n", str)
	}
	return
}

func StrCount(str, sub string, exp int) (s string) {
	count := strings.Count(str, sub)
	if count != exp {
		s = fmt.Sprintf("expected string to contain %d non-overlapping cases of substring\n", exp)
		s += bullet("count: %d\n", count)
	}
	return
}

func StrContainsFields(str string, fields []string) (s string) {
	set := make(map[string]struct{}, len(fields))
	for _, field := range strings.Fields(str) {
		set[field] = struct{}{}
	}
	var missing []string
	for _, field := range fields {
		if _, exists := set[field]; !exists {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		s = "expected fields of string to contain subset of values\n"
		s += bullet("missing: %s\n", strings.Join(missing, ", "))
	}
	return
}

func StrHasPrefix(prefix, str string) (s string) {
	if !strings.HasPrefix(str, prefix) {
		s = "expected string to have prefix\n"
		s += bullet("prefix: %s\n", prefix)
		s += bullet("string: %s\n", str)
	}
	return
}

func StrNotHasPrefix(prefix, str string) (s string) {
	if strings.HasPrefix(str, prefix) {
		s = "expected string to not have prefix; but it does\n"
		s += bullet("prefix: %s\n", prefix)
		s += bullet("string: %s\n", str)
	}
	return
}

func StrHasSuffix(suffix, str string) (s string) {
	if !strings.HasSuffix(str, suffix) {
		s = "expected string to have suffix\n"
		s += bullet("suffix: %s\n", suffix)
		s += bullet("string: %s\n", str)
	}
	return
}

func StrNotHasSuffix(suffix, str string) (s string) {
	if strings.HasSuffix(str, suffix) {
		s = "expected string to not have suffix; but it does\n"
		s += bullet("suffix: %s\n", suffix)
		s += bullet("string: %s\n", str)
	}
	return
}

func RegexMatch(re *regexp.Regexp, target string) (s string) {
	if !re.MatchString(target) {
		s = "expected regexp match\n"
		s += bullet(" regex: %s\n", re)
		s += bullet("string: %s\n", target)
	}
	return
}

func RegexpCompiles(expr string) (s string) {
	if _, err := regexp.Compile(expr); err != nil {
		s = "expected regular expression to compile\n"
		s += bullet("regex: %s\n", expr)
		s += bullet("error: %v\n", err)
	}
	return
}

func RegexpCompilesPOSIX(expr string) (s string) {
	if _, err := regexp.CompilePOSIX(expr); err != nil {
		s = "expected regular expression to compile (posix)\n"
		s += bullet("regex: %s\n", expr)
		s += bullet("error: %v\n", err)
	}
	return
}

// a10b173d-1427-432d-8a27-b12eada42feb
var uuid4Re = regexp.MustCompile(`^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}$`)

func UUIDv4(id string) (s string) {
	if !uuid4Re.MatchString(id) {
		s = "expected well-formed v4 UUID\n"
		s += bullet("format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX\n")
		s += bullet("actual: " + id + "\n")
	}
	return
}

func Length(n int, length interfaces.LengthFunc) (s string) {
	if l := length.Len(); l != n {
		s = "expected different length\n"
		s += bullet("  length: %d\n", l)
		s += bullet("expected: %d\n", n)
	}
	return
}

func Size(n int, size interfaces.SizeFunc) (s string) {
	if l := size.Size(); l != n {
		s = "expected different size\n"
		s += bullet("    size: %d\n", l)
		s += bullet("expected: %d\n", n)
	}
	return
}

func Empty(e interfaces.EmptyFunc) (s string) {
	if !e.Empty() {
		s = "expected to be empty, but was not\n"
	}
	return
}

func NotEmpty(e interfaces.EmptyFunc) (s string) {
	if e.Empty() {
		s = "expected to not be empty, but is\n"
	}
	return
}

func Contains[C any](i C, c interfaces.ContainsFunc[C]) (s string) {
	if !c.Contains(i) {
		s = "expected to contain element, but does not\n"
	}
	return
}

func ContainsSubset[C any](elements []C, container interfaces.ContainsFunc[C]) (s string) {
	for i := 0; i < len(elements); i++ {
		element := elements[i]
		if !container.Contains(element) {
			s = "expected to contain element, but does not\n"
			s += bullet("element: %v\n", element)
			return
		}
	}
	return
}

func NotContains[C any](i C, c interfaces.ContainsFunc[C]) (s string) {
	if c.Contains(i) {
		s = "expected not to contain element, but it does\n"
	}
	return
}

func Wait(wc *wait.Constraint) (s string) {
	err := wc.Run()
	if err != nil {
		s = "expected condition to pass within wait context\n"
		s += bullet("error: %v\n", err)
		// context info?
	}
	return
}

type Tweak[E interfaces.CopyEqual[E]] struct {
	Field string
	Apply interfaces.TweakFunc[E]
}

// StructEqual will apply each Tweak and assert E.Equal captures the modification.
func StructEqual[E interfaces.CopyEqual[E]](original E, tweaks []Tweak[E]) (s string) {
	for _, tweak := range tweaks {
		if tweak.Field == "" {
			return "Tweak.Field must be set"
		} else if tweak.Apply == nil {
			return "Tweak.Apply must be set"
		}
		clone := original.Copy()
		if s = Equal[E](original, clone); s != "" {
			return
		}
		tweak.Apply(clone)
		if s = NotEqual[E](original, clone); s != "" {
			return
		}
	}
	return
}

func bullet(msg string, args ...any) string {
	return fmt.Sprintf("↪ "+msg, args...)
}
