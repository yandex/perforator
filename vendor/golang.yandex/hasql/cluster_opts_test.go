/*
   Copyright 2020 YANDEX LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package hasql

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClusterDefaults(t *testing.T) {
	f := newFixture(t, 1)
	c, err := NewCluster(f.ClusterNodes(), f.PrimaryChecker)
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Close()) }()

	require.Equal(t, DefaultUpdateInterval, c.updateInterval)
	require.Equal(t, DefaultUpdateTimeout, c.updateTimeout)
}

func TestWithUpdateInterval(t *testing.T) {
	f := newFixture(t, 1)
	d := time.Hour
	c, err := NewCluster(f.ClusterNodes(), f.PrimaryChecker, WithUpdateInterval(d))
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Close()) }()

	require.Equal(t, d, c.updateInterval)
}

func TestWithUpdateTimeout(t *testing.T) {
	f := newFixture(t, 1)
	d := time.Hour
	c, err := NewCluster(f.ClusterNodes(), f.PrimaryChecker, WithUpdateTimeout(d))
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Close()) }()

	require.Equal(t, d, c.updateTimeout)
}

func TestWithNodePicker(t *testing.T) {
	var called bool
	picker := func([]Node) Node {
		called = true
		return nil
	}
	f := newFixture(t, 1)
	c, err := NewCluster(f.ClusterNodes(), f.PrimaryChecker, WithNodePicker(picker))
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Close()) }()

	c.picker(nil)
	require.True(t, called)
}

func TestWithTracer(t *testing.T) {
	var called int32
	tracer := Tracer{
		NotifiedWaiters: func() {
			atomic.StoreInt32(&called, 1)
		},
	}
	f := newFixture(t, 1)
	c, err := NewCluster(f.ClusterNodes(), f.PrimaryChecker, WithTracer(tracer))
	require.NoError(t, err)
	defer func() { require.NoError(t, c.Close()) }()

	c.tracer.NotifiedWaiters()
	require.Equal(t, int32(1), atomic.LoadInt32(&called))
}
