package genstack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requirePopEmpty[T any](t testing.TB, s *GenStack[T]) {
	v, ok := s.Pop()
	require.False(t, ok)
	require.Zero(t, v)
}

func requirePop[T any](t testing.TB, s *GenStack[T], expected T) {
	v, ok := s.Pop()
	require.True(t, ok)
	require.Equal(t, expected, v)
}

func TestGenStack_Empty(t *testing.T) {
	s := NewGenStack[int]()
	requirePopEmpty(t, s)
}

func TestGenStack_SingleGen(t *testing.T) {
	r := require.New(t)
	s := NewGenStack[int]()

	s.Push(1)
	s.Push(2)
	r.Equal(2, s.Len())

	requirePop(t, s, 2)
	requirePop(t, s, 1)
	requirePopEmpty(t, s)
}

func TestGenStack_TwoGen(t *testing.T) {
	r := require.New(t)
	s := NewGenStack[int]()

	s.Push(3)
	s.Push(4)
	s.Push(5)
	r.Equal(3, s.Len())
	s.NextGen()
	r.Equal(3, s.Len())
	s.Push(6)
	s.Push(7)
	r.Equal(5, s.Len())

	requirePop(t, s, 5)
	requirePop(t, s, 4)
	requirePop(t, s, 3)
	requirePop(t, s, 7)
	requirePop(t, s, 6)
	requirePopEmpty(t, s)
}

func TestGenStack_MuptiGen(t *testing.T) {
	r := require.New(t)
	s := NewGenStack[int]()

	s.Push(10)
	s.Push(11)
	s.Push(12)
	r.Equal(3, s.Len())
	s.NextGen()
	r.Equal(3, s.Len())
	s.Push(13)
	s.Push(14)
	r.Equal(5, s.Len())
	s.NextGen()
	r.Equal(5, s.Len())
	s.Push(15)
	s.Push(16)
	s.Push(17)
	r.Equal(8, s.Len())

	requirePop(t, s, 12)
	requirePop(t, s, 11)
	requirePop(t, s, 10)
	requirePop(t, s, 14)
	requirePop(t, s, 13)
	requirePop(t, s, 17)
	requirePop(t, s, 16)
	requirePop(t, s, 15)
	requirePopEmpty(t, s)
}
