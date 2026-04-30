package ptr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTOrNilInt(t *testing.T) {
	value5 := 5

	tests := []struct {
		name  string
		value int
		want  *int
	}{
		{
			name:  "zero",
			value: 0,
			want:  nil,
		},
		{
			name:  "not zero",
			value: value5,
			want:  &value5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := TOrNil(tt.value)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTOrNilStr(t *testing.T) {
	value5 := "5"

	tests := []struct {
		name  string
		value string
		want  *string
	}{
		{
			name:  "zero",
			value: "",
			want:  nil,
		},
		{
			name:  "not zero",
			value: value5,
			want:  &value5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := TOrNil(tt.value)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTOrNilStruct(t *testing.T) {
	type customStruct struct {
		A int64
		B string
	}

	value5 := customStruct{
		A: 5,
	}

	tests := []struct {
		name  string
		value customStruct
		want  *customStruct
	}{
		{
			name:  "zero",
			value: customStruct{},
			want:  nil,
		},
		{
			name:  "not zero",
			value: value5,
			want:  &value5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TOrNil(tt.value)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValueInt(t *testing.T) {
	value5 := 5

	tests := []struct {
		name string
		ptr  *int
		want int
	}{
		{
			name: "nil",
			ptr:  nil,
			want: 0,
		},
		{
			name: "not nil",
			ptr:  &value5,
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Value(tt.ptr)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValueString(t *testing.T) {
	valueHello := "hello"

	tests := []struct {
		name string
		ptr  *string
		want string
	}{
		{
			name: "nil",
			ptr:  nil,
			want: "",
		},
		{
			name: "not nil",
			ptr:  &valueHello,
			want: "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Value(tt.ptr)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValueOrInt(t *testing.T) {
	value5 := 5

	tests := []struct {
		name         string
		ptr          *int
		defaultValue int
		want         int
	}{
		{
			name:         "nil",
			ptr:          nil,
			defaultValue: 42,
			want:         42,
		},
		{
			name:         "not nil",
			ptr:          &value5,
			defaultValue: 42,
			want:         5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValueOr(tt.ptr, tt.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValueOrString(t *testing.T) {
	valueHello := "hello"

	tests := []struct {
		name         string
		ptr          *string
		defaultValue string
		want         string
	}{
		{
			name:         "nil",
			ptr:          nil,
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "not nil",
			ptr:          &valueHello,
			defaultValue: "default",
			want:         "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValueOr(tt.ptr, tt.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValueOrStruct(t *testing.T) {
	type customStruct struct {
		A int64
		B string
	}

	valueStruct := customStruct{A: 5, B: "test"}
	defaultStruct := customStruct{A: 100, B: "default"}

	tests := []struct {
		name         string
		ptr          *customStruct
		defaultValue customStruct
		want         customStruct
	}{
		{
			name:         "nil",
			ptr:          nil,
			defaultValue: defaultStruct,
			want:         defaultStruct,
		},
		{
			name:         "not nil",
			ptr:          &valueStruct,
			defaultValue: defaultStruct,
			want:         valueStruct,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValueOr(tt.ptr, tt.defaultValue)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualVal(t *testing.T) {
	type S struct {
		i int
		s string
	}

	tests := []struct {
		name string
		v    *S
		w    *S
		want bool
	}{
		{
			name: "equal nil",
			v:    nil,
			w:    nil,
			want: true,
		},
		{
			name: "v nil",
			v:    nil,
			w:    T(S{i: 0, s: "s"}),
			want: false,
		},
		{
			name: "w nil",
			v:    T(S{i: 9, s: "c"}),
			w:    nil,
			want: false,
		},
		{
			name: "equal non-nil",
			v:    T(S{i: 1, s: "a"}),
			w:    T(S{i: 1, s: "a"}),
			want: true,
		},
		{
			name: "not equal non-nil",
			v:    T(S{i: 1, s: "a"}),
			w:    T(S{i: 5, s: "a"}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EqualVal(tt.v, tt.w)

			assert.Equal(t, tt.want, got)
		})
	}

}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name string
		args []*int
		want *int
	}{
		{
			name: "empty",
			args: nil,
			want: nil,
		},
		{
			name: "only nil",
			args: []*int{nil, nil, nil},
			want: nil,
		},
		{
			name: "full",
			args: []*int{T(1), T(2), T(3)},
			want: T(1),
		},
		{
			name: "ignore nil",
			args: []*int{nil, nil, T(1), nil, T(2)},
			want: T(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Coalesce(tt.args...)

			assert.Equal(t, tt.want, got)
		})
	}
}
