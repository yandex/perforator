// Code generated via scripts/generate.sh. DO NOT EDIT.

// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

package must

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shoenig/test/interfaces"
	"github.com/shoenig/test/internal/assertions"
	"github.com/shoenig/test/internal/brokenfs"
	"github.com/shoenig/test/internal/constraints"
	"github.com/shoenig/test/internal/util"
	"github.com/shoenig/test/wait"
)

// Nil asserts a is nil.
func Nil(t T, a any, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Nil(a), settings...)
}

// NotNil asserts a is not nil.
func NotNil(t T, a any, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotNil(a), settings...)
}

// True asserts that condition is true.
func True(t T, condition bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.True(condition), settings...)
}

// False asserts condition is false.
func False(t T, condition bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.False(condition), settings...)
}

// Unreachable asserts a code path is not executed.
func Unreachable(t T, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Unreachable(), settings...)
}

// Error asserts err is a non-nil error.
func Error(t T, err error, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Error(err), settings...)
}

// EqError asserts err contains message msg.
func EqError(t T, err error, msg string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.EqError(err, msg), settings...)
}

// ErrorIs asserts err
func ErrorIs(t T, err error, target error, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.ErrorIs(err, target), settings...)
}

// NoError asserts err is a nil error.
func NoError(t T, err error, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NoError(err), settings...)
}

// ErrorContains asserts err contains sub.
func ErrorContains(t T, err error, sub string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.ErrorContains(err, sub), settings...)
}

// Eq asserts exp and val are equal using cmp.Equal.
func Eq[A any](t T, exp, val A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Eq(exp, val, options(settings...)...), settings...)
}

// EqOp asserts exp == val.
func EqOp[C comparable](t T, exp, val C, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.EqOp(exp, val), settings...)
}

// EqFunc asserts exp and val are equal using eq.
func EqFunc[A any](t T, exp, val A, eq func(a, b A) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.EqFunc(exp, val, eq), settings...)
}

// NotEq asserts exp and val are not equal using cmp.Equal.
func NotEq[A any](t T, exp, val A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotEq(exp, val, options(settings...)...), settings...)
}

// NotEqOp asserts exp != val.
func NotEqOp[C comparable](t T, exp, val C, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotEqOp(exp, val), settings...)
}

// NotEqFunc asserts exp and val are not equal using eq.
func NotEqFunc[A any](t T, exp, val A, eq func(a, b A) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotEqFunc(exp, val, eq), settings...)
}

// EqJSON asserts exp and val are equivalent JSON.
func EqJSON(t T, exp, val string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.EqJSON(exp, val), settings...)
}

// ValidJSON asserts js is valid JSON.
func ValidJSON(t T, js string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.ValidJSON(js), settings...)
}

// ValidJSONBytes asserts js is valid JSON.
func ValidJSONBytes(t T, js []byte, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.ValidJSONBytes(js))
}

// Equal asserts val.Equal(exp).
func Equal[E interfaces.EqualFunc[E]](t T, exp, val E, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Equal(exp, val), settings...)
}

// NotEqual asserts !val.Equal(exp).
func NotEqual[E interfaces.EqualFunc[E]](t T, exp, val E, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotEqual(exp, val), settings...)
}

// Lesser asserts val.Less(exp).
func Lesser[L interfaces.LessFunc[L]](t T, exp, val L, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Lesser(exp, val), settings...)
}

// SliceEqFunc asserts elements of val satisfy eq for the corresponding element in exp.
func SliceEqFunc[A, B any](t T, exp []B, val []A, eq func(expectation A, value B) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.EqSliceFunc(exp, val, eq), settings...)
}

// SliceEqual asserts val[n].Equal(exp[n]) for each element n.
func SliceEqual[E interfaces.EqualFunc[E]](t T, exp, val []E, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceEqual(exp, val), settings...)
}

// SliceEmpty asserts slice is empty.
func SliceEmpty[A any](t T, slice []A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceEmpty(slice), settings...)
}

// SliceNotEmpty asserts slice is not empty.
func SliceNotEmpty[A any](t T, slice []A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceNotEmpty(slice), settings...)
}

// SliceLen asserts slice is of length n.
func SliceLen[A any](t T, n int, slice []A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceLen(n, slice), settings...)
}

// Len asserts slice is of length n.
//
// Shorthand function for SliceLen. For checking Len() of a struct,
// use the Length() assertion.
func Len[A any](t T, n int, slice []A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceLen(n, slice), settings...)
}

// SliceContainsOp asserts item exists in slice using == operator.
func SliceContainsOp[C comparable](t T, slice []C, item C, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceContainsOp(slice, item), settings...)
}

// SliceContainsFunc asserts item exists in slice, using eq to compare elements.
func SliceContainsFunc[A, B any](t T, slice []A, item B, eq func(a A, b B) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceContainsFunc(slice, item, eq), settings...)
}

// SliceContainsEqual asserts item exists in slice, using Equal to compare elements.
func SliceContainsEqual[E interfaces.EqualFunc[E]](t T, slice []E, item E, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceContainsEqual(slice, item), settings...)
}

// SliceContains asserts item exists in slice, using cmp.Equal to compare elements.
func SliceContains[A any](t T, slice []A, item A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceContains(slice, item, options(settings...)...), settings...)
}

// SliceNotContains asserts item does not exist in slice, using cmp.Equal to
// compare elements.
func SliceNotContains[A any](t T, slice []A, item A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceNotContains(slice, item), settings...)
}

// SliceNotContainsFunc asserts item does not exist inslice, using eq to compare
// elements.
func SliceNotContainsFunc[A, B any](t T, slice []A, item B, eq func(a A, b B) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceNotContainsFunc(slice, item, eq), settings...)
}

// SliceContainsAll asserts slice and items contain the same elements, but in
// no particular order. The number of elements in slice and items must be the
// same.
func SliceContainsAll[A any](t T, slice, items []A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceContainsAll(slice, items, options(settings...)...), settings...)
}

// SliceContainsSubset asserts slice contains each item in items, in no particular
// order. There could be additional elements in slice not in items.
func SliceContainsSubset[A any](t T, slice, items []A, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.SliceContainsSubset(slice, items, options(settings...)...), settings...)
}

// Positive asserts n > 0.
func Positive[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Positive(n), settings...)
}

// NonPositive asserts n ≤ 0.
func NonPositive[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NonPositive(n), settings...)
}

// Negative asserts n < 0.
func Negative[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Negative(n), settings...)
}

// NonNegative asserts n >= 0.
func NonNegative[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NonNegative(n), settings...)
}

// Zero asserts n == 0.
func Zero[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Zero(n), settings...)
}

// NonZero asserts n != 0.
func NonZero[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NonZero(n), settings...)
}

// One asserts n == 1.
func One[N interfaces.Number](t T, n N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.One(n), settings...)
}

// Less asserts val < exp.
func Less[O constraints.Ordered](t T, exp, val O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Less(exp, val), settings...)
}

// LessEq asserts val ≤ exp.
func LessEq[O constraints.Ordered](t T, exp, val O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.LessEq(exp, val), settings...)
}

// Greater asserts val > exp.
func Greater[O constraints.Ordered](t T, exp, val O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Greater(exp, val), settings...)
}

// GreaterEq asserts val ≥ exp.
func GreaterEq[O constraints.Ordered](t T, exp, val O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.GreaterEq(exp, val), settings...)
}

// Between asserts lower ≤ val ≤ upper.
func Between[O constraints.Ordered](t T, lower, val, upper O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Between(lower, val, upper), settings...)
}

// BetweenExclusive asserts lower < val < upper.
func BetweenExclusive[O constraints.Ordered](t T, lower, val, upper O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.BetweenExclusive(lower, val, upper), settings...)
}

// Min asserts collection.Min() is equal to expect.
//
// The equality method may be configured with Cmp options.
func Min[A any, C interfaces.MinFunc[A]](t T, expect A, collection C, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Min(expect, collection, options(settings...)...), settings...)
}

// Max asserts collection.Max() is equal to expect.
//
// The equality method may be configured with Cmp options.
func Max[A any, C interfaces.MaxFunc[A]](t T, expect A, collection C, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Max(expect, collection, options(settings...)...), settings...)
}

// Ascending asserts slice[n] ≤ slice[n+1] for each element.
func Ascending[O constraints.Ordered](t T, slice []O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Ascending(slice), settings...)
}

// AscendingFunc asserts slice[n] is less than slice[n+1] for each element using the less comparator.
func AscendingFunc[A any](t T, slice []A, less func(A, A) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.AscendingFunc(slice, less), settings...)
}

// AscendingCmp asserts slice[n] is less than slice[n+1] for each element using the cmp comparator.
func AscendingCmp[A any](t T, slice []A, compare func(A, A) int, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.AscendingCmp(slice, compare), settings...)
}

// AscendingLess asserts slice[n].Less(slice[n+1]) for each element.
func AscendingLess[L interfaces.LessFunc[L]](t T, slice []L, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.AscendingLess(slice), settings...)
}

// Descending asserts slice[n] ≥ slice[n+1] for each element.
func Descending[O constraints.Ordered](t T, slice []O, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Descending(slice), settings...)
}

// DescendingFunc asserts slice[n+1] is less than slice[n] for each element using the less comparator.
func DescendingFunc[A any](t T, slice []A, less func(A, A) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.DescendingFunc(slice, less), settings...)
}

// DescendingCmp asserts slice[n+1] is ≤ slice[n] for each element.
func DescendingCmp[A any](t T, slice []A, compare func(A, A) int, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.DescendingCmp(slice, compare), settings...)
}

// DescendingLess asserts slice[n+1].Less(slice[n]) for each element.
func DescendingLess[L interfaces.LessFunc[L]](t T, slice []L, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.DescendingLess(slice), settings...)
}

// InDelta asserts a and b are within delta of each other.
func InDelta[N interfaces.Number](t T, a, b, delta N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.InDelta(a, b, delta), settings...)
}

// InDeltaSlice asserts each element a[n] is within delta of b[n].
func InDeltaSlice[N interfaces.Number](t T, a, b []N, delta N, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.InDeltaSlice(a, b, delta), settings...)
}

// MapEq asserts maps exp and val contain the same key/val pairs, using
// cmp.Equal function to compare vals.
func MapEq[M1, M2 interfaces.Map[K, V], K comparable, V any](t T, exp M1, val M2, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapEq(exp, val, options(settings...)), settings...)
}

// MapEqFunc asserts maps exp and val contain the same key/val pairs, using eq to
// compare vals.
func MapEqFunc[M1, M2 interfaces.Map[K, V], K comparable, V any](t T, exp M1, val M2, eq func(V, V) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapEqFunc(exp, val, eq), settings...)
}

// MapEqual asserts maps exp and val contain the same key/val pairs, using Equal
// method to compare vals
func MapEqual[M interfaces.MapEqualFunc[K, V], K comparable, V interfaces.EqualFunc[V]](t T, exp, val M, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapEqual(exp, val), settings...)
}

// MapLen asserts map is of size n.
func MapLen[M ~map[K]V, K comparable, V any](t T, n int, m M, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapLen(n, m), settings...)
}

// MapEmpty asserts map is empty.
func MapEmpty[M ~map[K]V, K comparable, V any](t T, m M, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapEmpty(m), settings...)
}

// MapNotEmpty asserts map is not empty.
func MapNotEmpty[M ~map[K]V, K comparable, V any](t T, m M, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapNotEmpty(m), settings...)
}

// MapContainsKey asserts m contains key.
func MapContainsKey[M ~map[K]V, K comparable, V any](t T, m M, key K, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapContainsKey(m, key), settings...)
}

// MapNotContainsKey asserts m does not contain key.
func MapNotContainsKey[M ~map[K]V, K comparable, V any](t T, m M, key K, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapNotContainsKey(m, key), settings...)
}

// MapContainsKeys asserts m contains each key in keys.
func MapContainsKeys[M ~map[K]V, K comparable, V any](t T, m M, keys []K, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapContainsKeys(m, keys), settings...)
}

// MapNotContainsKeys asserts m does not contain any key in keys.
func MapNotContainsKeys[M ~map[K]V, K comparable, V any](t T, m M, keys []K, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapNotContainsKeys(m, keys), settings...)
}

// MapContainsValues asserts m contains each val in vals.
func MapContainsValues[M ~map[K]V, K comparable, V any](t T, m M, vals []V, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapContainsValues(m, vals, options(settings...)), settings...)
}

// MapNotContainsValues asserts m does not contain any value in vals.
func MapNotContainsValues[M ~map[K]V, K comparable, V any](t T, m M, vals []V, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapNotContainsValues(m, vals, options(settings...)), settings...)
}

// MapContainsValuesFunc asserts m contains each val in vals using the eq function.
func MapContainsValuesFunc[M ~map[K]V, K comparable, V any](t T, m M, vals []V, eq func(V, V) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapContainsValuesFunc(m, vals, eq), settings...)
}

// MapNotContainsValuesFunc asserts m does not contain any value in vals using the eq function.
func MapNotContainsValuesFunc[M ~map[K]V, K comparable, V any](t T, m M, vals []V, eq func(V, V) bool, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapNotContainsValuesFunc(m, vals, eq), settings...)
}

// MapContainsValuesEqual asserts m contains each val in vals using the V.Equal method.
func MapContainsValuesEqual[M ~map[K]V, K comparable, V interfaces.EqualFunc[V]](t T, m M, vals []V, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapContainsValuesEqual(m, vals), settings...)
}

// MapNotContainsValuesEqual asserts m does not contain any value in vals using the V.Equal method.
func MapNotContainsValuesEqual[M ~map[K]V, K comparable, V interfaces.EqualFunc[V]](t T, m M, vals []V, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.MapNotContainsValuesEqual(m, vals), settings...)
}

// FileExistsFS asserts file exists on the fs.FS filesystem.
//
// Example,
// FileExistsFS(t, os.DirFS("/etc"), "hosts")
func FileExistsFS(t T, system fs.FS, file string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.FileExistsFS(system, file), settings...)
}

// FileExists asserts file exists on the OS filesystem.
func FileExists(t T, file string, settings ...Setting) {
	t.Helper()
	dir := filepath.Dir(file)
	file = filepath.Base(file)
	invoke(t, assertions.FileExistsFS(os.DirFS(dir), file), settings...)
}

// FileNotExistsFS asserts file does not exist on the fs.FS filesystem.
//
// Example,
// FileNotExist(t, os.DirFS("/bin"), "exploit.exe")
func FileNotExistsFS(t T, system fs.FS, file string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.FileNotExistsFS(system, file), settings...)
}

// FileNotExists asserts file does not exist on the OS filesystem.
func FileNotExists(t T, file string, settings ...Setting) {
	t.Helper()
	dir := filepath.Dir(file)
	file = filepath.Base(file)
	invoke(t, assertions.FileNotExistsFS(os.DirFS(dir), file), settings...)
}

// DirExistsFS asserts directory exists on the fs.FS filesystem.
//
// Example,
// DirExistsFS(t, os.DirFS("/usr/local"), "bin")
func DirExistsFS(t T, system fs.FS, directory string, settings ...Setting) {
	t.Helper()
	directory = strings.TrimPrefix(directory, "/")
	invoke(t, assertions.DirExistsFS(system, directory), settings...)
}

// DirExists asserts directory exists on the OS filesystem.
func DirExists(t T, directory string, settings ...Setting) {
	t.Helper()
	directory = strings.TrimPrefix(directory, "/")
	invoke(t, assertions.DirExistsFS(os.DirFS(brokenfs.Root), directory), settings...)
}

// DirNotExistsFS asserts directory does not exist on the fs.FS filesystem.
//
// Example,
// DirNotExistsFS(t, os.DirFS("/tmp"), "scratch")
func DirNotExistsFS(t T, system fs.FS, directory string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.DirNotExistsFS(system, directory), settings...)
}

// DirNotExists asserts directory does not exist on the OS filesystem.
func DirNotExists(t T, directory string, settings ...Setting) {
	t.Helper()
	directory = strings.TrimPrefix(directory, "/")
	invoke(t, assertions.DirNotExistsFS(os.DirFS(brokenfs.Root), directory), settings...)
}

// FileModeFS asserts the file or directory at path on fs.FS has exactly the given permission bits.
//
// Example,
// FileModeFS(t, os.DirFS("/bin"), "find", 0655)
func FileModeFS(t T, system fs.FS, path string, permissions fs.FileMode, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.FileModeFS(system, path, permissions), settings...)
}

// FileMode asserts the file or directory at path on the OS filesystem has exactly the given permission bits.
func FileMode(t T, path string, permissions fs.FileMode, settings ...Setting) {
	t.Helper()
	path = strings.TrimPrefix(path, "/")
	invoke(t, assertions.FileModeFS(os.DirFS(brokenfs.Root), path, permissions), settings...)
}

// FileContainsFS asserts the file on fs.FS contains content as a substring.
//
// Often os.DirFS is used to interact with the host filesystem.
// Example,
// FileContainsFS(t, os.DirFS("/etc"), "hosts", "localhost")
func FileContainsFS(t T, system fs.FS, file, content string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.FileContainsFS(system, file, content), settings...)
}

// FileContains asserts the file on the OS filesystem contains content as a substring.
func FileContains(t T, file, content string, settings ...Setting) {
	t.Helper()
	file = strings.TrimPrefix(file, "/")
	invoke(t, assertions.FileContainsFS(os.DirFS(brokenfs.Root), file, content), settings...)
}

// FilePathValid asserts path is a valid file path.
func FilePathValid(t T, path string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.FilePathValid(path), settings...)
}

// Close asserts c.Close does not cause an error.
func Close(t T, c io.Closer) {
	t.Helper()
	invoke(t, assertions.Close(c))
}

// StrEqFold asserts exp and val are equivalent, ignoring case.
func StrEqFold(t T, exp, val string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrEqFold(exp, val), settings...)
}

// StrNotEqFold asserts exp and val are not equivalent, ignoring case.
func StrNotEqFold(t T, exp, val string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrNotEqFold(exp, val), settings...)
}

// StrContains asserts s contains substring sub.
func StrContains(t T, s, sub string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrContains(s, sub), settings...)
}

// StrContainsFold asserts s contains substring sub, ignoring case.
func StrContainsFold(t T, s, sub string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrContainsFold(s, sub), settings...)
}

// StrNotContains asserts s does not contain substring sub.
func StrNotContains(t T, s, sub string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrNotContains(s, sub), settings...)
}

// StrNotContainsFold asserts s does not contain substring sub, ignoring case.
func StrNotContainsFold(t T, s, sub string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrNotContainsFold(s, sub), settings...)
}

// StrContainsAny asserts s contains at least one character in chars.
func StrContainsAny(t T, s, chars string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrContainsAny(s, chars), settings...)
}

// StrNotContainsAny asserts s does not contain any character in chars.
func StrNotContainsAny(t T, s, chars string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrNotContainsAny(s, chars), settings...)
}

// StrCount asserts s contains exactly count instances of substring sub.
func StrCount(t T, s, sub string, count int, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrCount(s, sub, count), settings...)
}

// StrContainsFields asserts that fields is a subset of the result of strings.Fields(s).
func StrContainsFields(t T, s string, fields []string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrContainsFields(s, fields), settings...)
}

// StrHasPrefix asserts that s starts with prefix.
func StrHasPrefix(t T, prefix, s string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrHasPrefix(prefix, s), settings...)
}

// StrNotHasPrefix asserts that s does not start with prefix.
func StrNotHasPrefix(t T, prefix, s string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrNotHasPrefix(prefix, s), settings...)
}

// StrHasSuffix asserts that s ends with suffix.
func StrHasSuffix(t T, suffix, s string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrHasSuffix(suffix, s), settings...)
}

// StrNotHasSuffix asserts that s does not end with suffix.
func StrNotHasSuffix(t T, suffix, s string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StrNotHasSuffix(suffix, s), settings...)
}

// RegexMatch asserts regular expression re matches string s.
func RegexMatch(t T, re *regexp.Regexp, s string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.RegexMatch(re, s), settings...)
}

// RegexCompiles asserts expr compiles as a valid regular expression.
func RegexCompiles(t T, expr string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.RegexpCompiles(expr), settings...)
}

// RegexCompilesPOSIX asserts expr compiles as a valid POSIX regular expression.
func RegexCompilesPOSIX(t T, expr string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.RegexpCompilesPOSIX(expr), settings...)
}

// UUIDv4 asserts id meets the criteria of a v4 UUID.
func UUIDv4(t T, id string, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.UUIDv4(id), settings...)
}

// Size asserts s.Size() is equal to exp.
func Size(t T, exp int, s interfaces.SizeFunc, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Size(exp, s), settings...)
}

// Length asserts l.Len() is equal to exp.
func Length(t T, exp int, l interfaces.LengthFunc, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Length(exp, l), settings...)
}

// Empty asserts e.Empty() is true.
func Empty(t T, e interfaces.EmptyFunc, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Empty(e), settings...)
}

// NotEmpty asserts e.Empty() is false.
func NotEmpty(t T, e interfaces.EmptyFunc, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotEmpty(e), settings...)
}

// Contains asserts container.ContainsFunc(element) is true.
func Contains[C any](t T, element C, container interfaces.ContainsFunc[C], settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Contains(element, container), settings...)
}

// ContainsSubset asserts each element in elements exists in container, in no particular order.
// There may be elements in container beyond what is present in elements.
func ContainsSubset[C any](t T, elements []C, container interfaces.ContainsFunc[C], settings ...Setting) {
	t.Helper()
	invoke(t, assertions.ContainsSubset(elements, container), settings...)
}

// NotContains asserts container.ContainsFunc(element) is false.
func NotContains[C any](t T, element C, container interfaces.ContainsFunc[C], settings ...Setting) {
	t.Helper()
	invoke(t, assertions.NotContains(element, container), settings...)
}

// Wait asserts wc.
func Wait(t T, wc *wait.Constraint, settings ...Setting) {
	t.Helper()
	invoke(t, assertions.Wait(wc), settings...)
}

// Tweak is used to modify a struct and assert its Equal method captures the
// modification.
//
// Field is the name of the struct field and is used only for error printing.
// Apply is a function that modifies E.
type Tweak[E interfaces.CopyEqual[E]] struct {
	Field string
	Apply interfaces.TweakFunc[E]
}

// Tweaks is a slice of Tweak.
type Tweaks[E interfaces.CopyEqual[E]] []Tweak[E]

// StructEqual will apply each Tweak and assert E.Equal captures the modification.
func StructEqual[E interfaces.CopyEqual[E]](t T, original E, tweaks Tweaks[E], settings ...Setting) {
	t.Helper()
	invoke(t, assertions.StructEqual(
		original,
		util.CloneSliceFunc(
			tweaks,
			func(tweak Tweak[E]) assertions.Tweak[E] {
				return assertions.Tweak[E]{Field: tweak.Field, Apply: tweak.Apply}
			},
		),
	), settings...)
}
