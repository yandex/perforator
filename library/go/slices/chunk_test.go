package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/slices"
)

func TestChunkSlice(t *testing.T) {
	type args struct {
		slice     []int
		chunkSize int
	}
	type testCase struct {
		name string
		args args
		want [][]int
	}
	tests := []testCase{
		{
			name: "empty slice",
			args: args{slice: []int{}, chunkSize: 10},
			want: [][]int{},
		},
		{
			name: "empty slice, zero chunk size",
			args: args{slice: []int{}, chunkSize: 0},
			want: [][]int{{}},
		},
		{
			name: "zero chunk size",
			args: args{slice: []int{1, 2}, chunkSize: 0},
			want: [][]int{{1, 2}},
		},
		{
			name: "chunk size 1",
			args: args{slice: []int{1, 2, 3}, chunkSize: 1},
			want: [][]int{{1}, {2}, {3}},
		},
		{
			name: "chunk size 3, 3 elements",
			args: args{slice: []int{1, 2, 3}, chunkSize: 3},
			want: [][]int{{1, 2, 3}},
		},
		{
			name: "chunk size 3, 10 elements",
			args: args{slice: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, chunkSize: 3},
			want: [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10}},
		},
		{
			name: "chunk size 3, 2 elements",
			args: args{slice: []int{1, 2}, chunkSize: 3},
			want: [][]int{{1, 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, slices.Chunk(tt.args.slice, tt.args.chunkSize), "Chunk(%v, %v)", tt.args.slice, tt.args.chunkSize)
		})
	}
}
