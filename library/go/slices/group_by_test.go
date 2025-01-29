package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func toUniqueResult[T any, K comparable](m map[K][]T) map[K]T {
	res := map[K]T{}

	for key := range m {
		res[key] = m[key][0]
	}

	return res
}

// region Test cases

var (
	valString       = []string{"1", "2", "4", "3"}
	keyGetterString = func(t string) string { return t }
	expectedString  = map[string][]string{
		"1": {"1"},
		"2": {"2"},
		"3": {"3"},
		"4": {"4"},
	}
	expectedStringUnique  = toUniqueResult(expectedString)
	expectedStringIndexed = map[string][]slices.IndexedEntity[string]{
		"1": {{"1", 0}},
		"2": {{"2", 1}},
		"3": {{"3", 3}},
		"4": {{"4", 2}},
	}
	expectedStringUniqueIndexed = toUniqueResult(expectedStringIndexed)
)

var (
	valInt       = []int{1, 2, 4, 3}
	keyGetterInt = func(t int) int { return t }
	expectedInt  = map[int][]int{
		1: {1},
		2: {2},
		3: {3},
		4: {4},
	}
	expectedIntUnique  = toUniqueResult(expectedInt)
	expectedIntIndexed = map[int][]slices.IndexedEntity[int]{
		1: {{1, 0}},
		2: {{2, 1}},
		3: {{3, 3}},
		4: {{4, 2}},
	}
	expectedIntUniqueIndexed = toUniqueResult(expectedIntIndexed)
)

var (
	valBool       = []bool{true, true, false, false, true}
	keyGetterBool = func(t bool) bool { return t }
	expectedBool  = map[bool][]bool{
		true:  {true, true, true},
		false: {false, false},
	}
	expectedBoolIndexed = map[bool][]slices.IndexedEntity[bool]{
		true:  {{true, 0}, {true, 1}, {true, 4}},
		false: {{false, 2}, {false, 3}},
	}

	valBoolUnique       = []bool{true, false}
	keyGetterBoolUnique = func(t bool) bool { return t }
	expectedBoolUnique  = map[bool]bool{
		true:  true,
		false: false,
	}
	expectedBoolUniqueIndexed = map[bool]slices.IndexedEntity[bool]{
		true:  {true, 0},
		false: {false, 1},
	}
)

type testStruct struct {
	Key   string
	Value int
}

var (
	valStruct = []testStruct{
		{Key: "a", Value: 4},
		{Key: "b", Value: 1},
		{Key: "a", Value: 3},
		{Key: "a", Value: 2},
	}
	keyGetterStruct = func(t testStruct) string { return t.Key }
	expectedStruct  = map[string][]testStruct{
		"a": {
			{Key: "a", Value: 4},
			{Key: "a", Value: 3},
			{Key: "a", Value: 2},
		},
		"b": {
			{Key: "b", Value: 1},
		},
	}
	expectedStructIndexed = map[string][]slices.IndexedEntity[testStruct]{
		"a": {
			{testStruct{Key: "a", Value: 4}, 0},
			{testStruct{Key: "a", Value: 3}, 2},
			{testStruct{Key: "a", Value: 2}, 3},
		},
		"b": {
			{testStruct{Key: "b", Value: 1}, 1},
		},
	}

	valStructUnique = []testStruct{
		{Key: "a", Value: 4},
		{Key: "b", Value: 1},
		{Key: "c", Value: 3},
		{Key: "d", Value: 2},
	}
	keyGetterStructUnique = func(t testStruct) string { return t.Key }
	expectedStructUnique  = map[string]testStruct{
		"a": {Key: "a", Value: 4},
		"b": {Key: "b", Value: 1},
		"c": {Key: "c", Value: 3},
		"d": {Key: "d", Value: 2},
	}
	expectedStructUniqueIndexed = map[string]slices.IndexedEntity[testStruct]{
		"a": {testStruct{Key: "a", Value: 4}, 0},
		"b": {testStruct{Key: "b", Value: 1}, 1},
		"c": {testStruct{Key: "c", Value: 3}, 2},
		"d": {testStruct{Key: "d", Value: 2}, 3},
	}
)

// endregion Test cases

// region GroupBy

func TestGroupByEmpty(t *testing.T) {
	assert.Equal(t, map[string][]string{}, slices.GroupBy([]string{}, keyGetterString))
}

func TestGroupByString(t *testing.T) {
	assert.Equal(t, expectedString, slices.GroupBy(valString, keyGetterString))
}

func TestGroupByInt(t *testing.T) {
	assert.Equal(t, expectedInt, slices.GroupBy(valInt, keyGetterInt))
}

func TestGroupByBool(t *testing.T) {
	assert.Equal(t, expectedBool, slices.GroupBy(valBool, keyGetterBool))
}

func TestGroupByStruct(t *testing.T) {
	assert.Equal(t, expectedStruct, slices.GroupBy(valStruct, keyGetterStruct))
}

// endregion GroupBy

// region GroupByUniqueKey

func TestGroupByUniqueKeyEmpty(t *testing.T) {
	res, err := slices.GroupByUniqueKey([]string{}, keyGetterString)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{}, res)
}

func TestGroupByUniqueKeyString(t *testing.T) {
	res, err := slices.GroupByUniqueKey(valString, keyGetterString)
	assert.NoError(t, err)
	assert.Equal(t, expectedStringUnique, res)
}

func TestGroupByUniqueKeyInt(t *testing.T) {
	res, err := slices.GroupByUniqueKey(valInt, keyGetterInt)
	assert.NoError(t, err)
	assert.Equal(t, expectedIntUnique, res)
}

func TestGroupByUniqueKeyBoolFail(t *testing.T) {
	_, err := slices.GroupByUniqueKey(valBool, keyGetterBool)
	assert.Error(t, err)
}

func TestGroupByUniqueKeyBoolSuccess(t *testing.T) {
	res, err := slices.GroupByUniqueKey(valBoolUnique, keyGetterBoolUnique)
	assert.NoError(t, err)
	assert.Equal(t, expectedBoolUnique, res)
}

func TestGroupByUniqueKeyStructFail(t *testing.T) {
	_, err := slices.GroupByUniqueKey(valStruct, keyGetterStruct)
	assert.Error(t, err)
}

func TestGroupByUniqueKeyStructSuccess(t *testing.T) {
	res, err := slices.GroupByUniqueKey(valStructUnique, keyGetterStructUnique)
	assert.NoError(t, err)
	assert.Equal(t, expectedStructUnique, res)
}

// endregion GroupByUniqueKey

// region GroupByWithIndex

func TestGroupByWithIndexEmpty(t *testing.T) {
	assert.Equal(t, map[string][]slices.IndexedEntity[string]{}, slices.GroupByWithIndex([]string{}, keyGetterString))
}

func TestGroupByWithIndexString(t *testing.T) {
	assert.Equal(t, expectedStringIndexed, slices.GroupByWithIndex(valString, keyGetterString))
}

func TestGroupByWithIndexInt(t *testing.T) {
	assert.Equal(t, expectedIntIndexed, slices.GroupByWithIndex(valInt, keyGetterInt))
}

func TestGroupByWithIndexBool(t *testing.T) {
	assert.Equal(t, expectedBoolIndexed, slices.GroupByWithIndex(valBool, keyGetterBool))
}

func TestGroupByWithIndexStruct(t *testing.T) {
	assert.Equal(t, expectedStructIndexed, slices.GroupByWithIndex(valStruct, keyGetterStruct))
}

// endregion GroupByWithIndex

// region GroupByUniqueKeyWithIndex

func TestGroupByUniqueKeyWithIndexEmpty(t *testing.T) {
	res, err := slices.GroupByUniqueKeyWithIndex([]string{}, keyGetterString)
	assert.NoError(t, err)
	assert.Equal(t, map[string]slices.IndexedEntity[string]{}, res)
}

func TestGroupByUniqueKeyWithIndexString(t *testing.T) {
	res, err := slices.GroupByUniqueKeyWithIndex(valString, keyGetterString)
	assert.NoError(t, err)
	assert.Equal(t, expectedStringUniqueIndexed, res)
}

func TestGroupByUniqueKeyWithIndexInt(t *testing.T) {
	res, err := slices.GroupByUniqueKeyWithIndex(valInt, keyGetterInt)
	assert.NoError(t, err)
	assert.Equal(t, expectedIntUniqueIndexed, res)
}

func TestGroupByUniqueKeyWithIndexBoolFail(t *testing.T) {
	_, err := slices.GroupByUniqueKeyWithIndex(valBool, keyGetterBool)
	assert.Error(t, err)
}

func TestGroupByUniqueKeyWithIndexBoolSuccess(t *testing.T) {
	res, err := slices.GroupByUniqueKeyWithIndex(valBoolUnique, keyGetterBoolUnique)
	assert.NoError(t, err)
	assert.Equal(t, expectedBoolUniqueIndexed, res)
}

func TestGroupByUniqueKeyWithIndexStructFail(t *testing.T) {
	_, err := slices.GroupByUniqueKeyWithIndex(valStruct, keyGetterStruct)
	assert.Error(t, err)
}

func TestGroupByUniqueKeyWithIndexStructSuccess(t *testing.T) {
	res, err := slices.GroupByUniqueKeyWithIndex(valStructUnique, keyGetterStructUnique)
	assert.NoError(t, err)
	assert.Equal(t, expectedStructUniqueIndexed, res)
}

// endregion GroupByUniqueKeyWithIndex
