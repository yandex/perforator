package unique_test

import (
	"fmt"
	"sort"

	"github.com/mpvl/unique"
)

func ExampleInts() {
	a := []int{1, 2, 2, 3, 3, 3}
	unique.Ints(&a)
	fmt.Println(a)

	// Output:
	// [1 2 3]
}

func ExampleFloat64s() {
	a := []float64{1.1, 2.2, 2.2, 3.3, 3.3, 3.3}
	unique.Float64s(&a)
	fmt.Println(a)

	// Output:
	// [1.1 2.2 3.3]
}

func ExampleStrings() {
	a := []string{"1", "2", "2", "3", "3", "3"}
	unique.Strings(&a)
	fmt.Println(a)

	// Output:
	// [1 2 3]
}

func ExampleSort() {
	a := []int{3, 3, 2, 3, 2, 1}
	unique.Sort(unique.IntSlice{&a})
	fmt.Println(a)

	// Output:
	// [1 2 3]
}

func ExampleIsUniqued() {
	fmt.Println(unique.IsUniqued(sort.IntSlice([]int{1, 2, 3})))
	fmt.Println(unique.IsUniqued(sort.IntSlice([]int{1, 2, 2, 3})))
	// false because it isn't sorted as well.
	fmt.Println(unique.IsUniqued(sort.IntSlice([]int{3, 2, 1})))
	fmt.Println(unique.IsUniqued(sort.IntSlice([]int{})))

	// Output:
	// true
	// false
	// false
	// true
}

func ExampleToFront() {
	a := []int{1, 2, 2, 3, 3, 3}
	fmt.Println(a[:unique.ToFront(sort.IntSlice(a))])

	// Output:
	// [1 2 3]
}
