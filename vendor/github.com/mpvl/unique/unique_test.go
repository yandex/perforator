package unique

import (
	"reflect"
	"testing"
)

func TestUnique(t *testing.T) {
	for i, tc := range []struct {
		data, want Interface
	}{
		{
			IntSlice{&[]int{}},
			IntSlice{&[]int{}},
		},
		{
			IntSlice{&[]int{1}},
			IntSlice{&[]int{1}},
		},
		{
			IntSlice{&[]int{1, 2}},
			IntSlice{&[]int{1, 2}},
		},
		{
			IntSlice{&[]int{1, 1}},
			IntSlice{&[]int{1}},
		},
		{
			IntSlice{&[]int{1, 2, 2}},
			IntSlice{&[]int{1, 2}},
		},
		{
			IntSlice{&[]int{1, 1, 2}},
			IntSlice{&[]int{1, 2}},
		},
	} {
		Unique(tc.data)
		if !reflect.DeepEqual(tc.data, tc.want) {
			t.Errorf("%d: got %v; want %v", i, tc.data, tc.want)
		}
	}
}
