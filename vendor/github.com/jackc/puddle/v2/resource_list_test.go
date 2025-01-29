package puddle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResList_Append(t *testing.T) {
	r := require.New(t)

	arr := []*Resource[any]{
		new(Resource[any]),
		new(Resource[any]),
		new(Resource[any]),
	}

	list := resList[any](arr)

	list.append(new(Resource[any]))
	r.Len(list, 4)
	list.append(new(Resource[any]))
	r.Len(list, 5)
	list.append(new(Resource[any]))
	r.Len(list, 6)
}

func TestResList_PopBack(t *testing.T) {
	r := require.New(t)

	arr := []*Resource[any]{
		new(Resource[any]),
		new(Resource[any]),
		new(Resource[any]),
	}

	list := resList[any](arr)

	list.popBack()
	r.Len(list, 2)
	list.popBack()
	r.Len(list, 1)
	list.popBack()
	r.Len(list, 0)

	r.Panics(func() { list.popBack() })
}

func TestResList_PanicsWithBugReportIfResourceDoesNotExist(t *testing.T) {
	arr := []*Resource[any]{
		new(Resource[any]),
		new(Resource[any]),
		new(Resource[any]),
	}

	list := resList[any](arr)

	assert.PanicsWithValue(t, "BUG: removeResource could not find res in slice", func() {
		list.remove(new(Resource[any]))
	})
}
