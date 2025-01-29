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
