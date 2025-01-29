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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCluster(t *testing.T) {
	fakeDB, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	require.NotNil(t, fakeDB)

	inputs := []struct {
		Name    string
		Fixture *fixture
		Err     bool
	}{
		{
			Name:    "no nodes",
			Fixture: newFixture(t, 0),
			Err:     true,
		},
		{
			Name:    "invalid node (no address)",
			Fixture: &fixture{Nodes: []*mockedNode{{Node: NewNode("", fakeDB)}}},
			Err:     true,
		},
		{
			Name:    "invalid node (no db)",
			Fixture: &fixture{Nodes: []*mockedNode{{Node: NewNode("fake.addr", nil)}}},
			Err:     true,
		},
		{
			Name:    "valid node",
			Fixture: newFixture(t, 1),
		},
	}

	for _, input := range inputs {
		t.Run(input.Name, func(t *testing.T) {
			defer input.Fixture.AssertExpectations(t)

			cl, err := NewCluster(input.Fixture.ClusterNodes(), input.Fixture.PrimaryChecker)
			if input.Err {
				require.Error(t, err)
				require.Nil(t, cl)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cl)
			defer func() { require.NoError(t, cl.Close()) }()

			require.Len(t, cl.Nodes(), len(input.Fixture.Nodes))
		})
	}
}

const (
	// How often nodes are updated in the background
	updateInterval = time.Millisecond * 20
	// Timeout of one WaitFor* iteration
	// Being half of updateInterval allows for hitting updates during wait and not hitting them at all
	waitTimeout = updateInterval / 2
)

func setupCluster(t *testing.T, f *fixture, tracer Tracer) *Cluster {
	cl, err := NewCluster(
		f.ClusterNodes(),
		f.PrimaryChecker,
		WithUpdateInterval(updateInterval),
		WithTracer(tracer),
	)
	require.NoError(t, err)
	require.NotNil(t, cl)
	require.Len(t, cl.Nodes(), len(f.Nodes))
	return cl
}

func waitForNode(t *testing.T, o *nodeUpdateObserver, wait func(ctx context.Context) (Node, error), expected Node) {
	o.StartObservation()

	var node Node
	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)

		node, err = wait(ctx)
		if o.ObservedUpdates() {
			cancel()
			break
		}

		cancel()
	}

	if expected != nil {
		require.NoError(t, err)
		require.Equal(t, expected, node)
	} else {
		require.Error(t, err)
		require.Nil(t, node)
	}
}

func waitForOneOfNodes(t *testing.T, o *nodeUpdateObserver, wait func(ctx context.Context) (Node, error), expected []Node) {
	o.StartObservation()

	var node Node
	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)

		node, err = wait(ctx)
		if o.ObservedUpdates() {
			cancel()
			break
		}

		cancel()
	}

	require.NoError(t, err)
	require.NotNil(t, node)

	for _, n := range expected {
		if n == node {
			return
		}
	}

	t.Fatalf("received node %+v but expected one of %+v", node, expected)
}

func TestCluster_WaitForAlive(t *testing.T) {
	inputs := []struct {
		Name    string
		Fixture *fixture
		Test    func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus)
	}{
		{
			Name:    "Alive",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus) {
				f.Nodes[0].setStatus(status)
				waitForNode(t, o, cl.WaitForAlive, f.Nodes[0].Node)
			},
		},
		{
			Name:    "Dead",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus) {
				waitForNode(t, o, cl.WaitForAlive, nil)
			},
		},
		{
			Name:    "AliveDeadAlive",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus) {
				node := f.Nodes[0]

				node.setStatus(status)
				waitForNode(t, o, cl.WaitForAlive, node.Node)

				node.setStatus(nodeStatusUnknown)
				waitForNode(t, o, cl.WaitForAlive, nil)

				node.setStatus(status)
				waitForNode(t, o, cl.WaitForAlive, node.Node)
			},
		},
		{
			Name:    "DeadAliveDead",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus) {
				node := f.Nodes[0]

				waitForNode(t, o, cl.WaitForAlive, nil)

				node.setStatus(status)
				waitForNode(t, o, cl.WaitForAlive, node.Node)

				node.setStatus(nodeStatusUnknown)
				waitForNode(t, o, cl.WaitForAlive, nil)
			},
		},
		{
			Name:    "AllAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus) {
				f.Nodes[0].setStatus(status)
				f.Nodes[1].setStatus(status)
				f.Nodes[2].setStatus(status)

				waitForOneOfNodes(t, o, cl.WaitForAlive, []Node{f.Nodes[0].Node, f.Nodes[1].Node, f.Nodes[2].Node})
			},
		},
		{
			Name:    "AllDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster, status nodeStatus) {
				waitForNode(t, o, cl.WaitForAlive, nil)
			},
		},
	}

	for _, status := range []nodeStatus{nodeStatusPrimary, nodeStatusStandby} {
		for _, input := range inputs {
			t.Run(fmt.Sprintf("%s status %d", input.Name, status), func(t *testing.T) {
				defer input.Fixture.AssertExpectations(t)

				var o nodeUpdateObserver
				cl := setupCluster(t, input.Fixture, o.Tracer())
				defer func() { require.NoError(t, cl.Close()) }()

				input.Test(t, input.Fixture, &o, cl, status)
			})
		}
	}
}

func TestCluster_WaitForPrimary(t *testing.T) {
	inputs := []struct {
		Name    string
		Fixture *fixture
		Test    func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster)
	}{
		{
			Name:    "PrimaryAlive",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForPrimary, f.Nodes[0].Node)
			},
		},
		{
			Name:    "PrimaryDead",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForPrimary, nil)
			},
		},
		{
			Name:    "AllAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[1].setStatus(nodeStatusPrimary)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForPrimary, f.Nodes[1].Node)
			},
		},
		{
			Name:    "AllDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForPrimary, nil)
			},
		},
		{
			Name:    "PrimaryAliveOtherDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForPrimary, f.Nodes[1].Node)
			},
		},
		{
			Name:    "PrimaryDeadOtherAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForPrimary, nil)
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.Name, func(t *testing.T) {
			defer input.Fixture.AssertExpectations(t)

			var o nodeUpdateObserver
			cl := setupCluster(t, input.Fixture, o.Tracer())
			defer func() { require.NoError(t, cl.Close()) }()

			input.Test(t, input.Fixture, &o, cl)
		})
	}
}

func TestCluster_WaitForStandby(t *testing.T) {
	inputs := []struct {
		Name    string
		Fixture *fixture
		Test    func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster)
	}{
		{
			Name:    "StandbyAlive",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForStandby, f.Nodes[0].Node)
			},
		},
		{
			Name:    "StandbyDead",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForStandby, nil)
			},
		},
		{
			Name:    "AllAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[1].setStatus(nodeStatusPrimary)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForOneOfNodes(t, o, cl.WaitForStandby, []Node{f.Nodes[0].Node, f.Nodes[2].Node})
			},
		},
		{
			Name:    "AllDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForStandby, nil)
			},
		},
		{
			Name:    "StandbyAliveOtherDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForStandby, f.Nodes[1].Node)
			},
		},
		{
			Name:    "StandbysAliveOtherDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusStandby)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForOneOfNodes(t, o, cl.WaitForStandby, []Node{f.Nodes[1].Node, f.Nodes[2].Node})
			},
		},
		{
			Name:    "StandbyDeadOtherAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusPrimary)
				f.Nodes[2].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForStandby, nil)
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.Name, func(t *testing.T) {
			defer input.Fixture.AssertExpectations(t)

			var o nodeUpdateObserver
			cl := setupCluster(t, input.Fixture, o.Tracer())
			defer func() { require.NoError(t, cl.Close()) }()

			input.Test(t, input.Fixture, &o, cl)
		})
	}
}

func TestCluster_WaitForPrimaryPreferred(t *testing.T) {
	inputs := []struct {
		Name    string
		Fixture *fixture
		Test    func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster)
	}{
		{
			Name:    "PrimaryAlive",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForPrimaryPreferred, f.Nodes[0].Node)
			},
		},
		{
			Name:    "PrimaryDead",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForPrimaryPreferred, nil)
			},
		},
		{
			Name:    "AllAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[1].setStatus(nodeStatusPrimary)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForPrimaryPreferred, f.Nodes[1].Node)
			},
		},
		{
			Name:    "AllDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForPrimary, nil)
			},
		},
		{
			Name:    "PrimaryAliveOtherDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForPrimaryPreferred, f.Nodes[1].Node)
			},
		},
		{
			Name:    "PrimaryDeadOtherAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForOneOfNodes(t, o, cl.WaitForPrimaryPreferred, []Node{f.Nodes[0].Node, f.Nodes[2].Node})
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.Name, func(t *testing.T) {
			defer input.Fixture.AssertExpectations(t)

			var o nodeUpdateObserver
			cl := setupCluster(t, input.Fixture, o.Tracer())
			defer func() { require.NoError(t, cl.Close()) }()

			input.Test(t, input.Fixture, &o, cl)
		})
	}
}

func TestCluster_WaitForStandbyPreferred(t *testing.T) {
	inputs := []struct {
		Name    string
		Fixture *fixture
		Test    func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster)
	}{
		{
			Name:    "StandbyAlive",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForStandbyPreferred, f.Nodes[0].Node)
			},
		},
		{
			Name:    "StandbyDead",
			Fixture: newFixture(t, 1),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForStandbyPreferred, nil)
			},
		},
		{
			Name:    "AllAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[1].setStatus(nodeStatusPrimary)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForOneOfNodes(t, o, cl.WaitForStandbyPreferred, []Node{f.Nodes[0].Node, f.Nodes[2].Node})
			},
		},
		{
			Name:    "AllDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForStandbyPreferred, nil)
			},
		},
		{
			Name:    "StandbyAliveOtherDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForStandbyPreferred, f.Nodes[1].Node)
			},
		},
		{
			Name:    "StandbysAliveOtherDead",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusStandby)
				f.Nodes[2].setStatus(nodeStatusStandby)
				waitForOneOfNodes(t, o, cl.WaitForStandbyPreferred, []Node{f.Nodes[1].Node, f.Nodes[2].Node})
			},
		},
		{
			Name:    "StandbyDeadOtherAlive",
			Fixture: newFixture(t, 3),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusPrimary)
				f.Nodes[2].setStatus(nodeStatusPrimary)
				waitForOneOfNodes(t, o, cl.WaitForStandbyPreferred, []Node{f.Nodes[0].Node, f.Nodes[2].Node})
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.Name, func(t *testing.T) {
			defer input.Fixture.AssertExpectations(t)

			var o nodeUpdateObserver
			cl := setupCluster(t, input.Fixture, o.Tracer())
			defer func() { require.NoError(t, cl.Close()) }()

			input.Test(t, input.Fixture, &o, cl)
		})
	}
}

func TestCluster_Err(t *testing.T) {
	inputs := []struct {
		Name    string
		Fixture *fixture
		Test    func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster)
	}{
		{
			Name:    "AllAlive",
			Fixture: newFixture(t, 2),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				f.Nodes[1].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForPrimary, f.Nodes[1].Node)

				require.NoError(t, cl.Err())
			},
		},
		{
			Name:    "AllDead",
			Fixture: newFixture(t, 2),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				waitForNode(t, o, cl.WaitForPrimary, nil)

				err := cl.Err()
				require.Error(t, err)
				assert.ErrorContains(t, err, fmt.Sprintf("%q node error occurred at", f.Nodes[0].Node.Addr()))
				assert.ErrorContains(t, err, fmt.Sprintf("%q node error occurred at", f.Nodes[1].Node.Addr()))
			},
		},
		{
			Name:    "PrimaryAliveOtherDead",
			Fixture: newFixture(t, 2),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[1].setStatus(nodeStatusPrimary)
				waitForNode(t, o, cl.WaitForPrimary, f.Nodes[1].Node)

				err := cl.Err()
				require.Error(t, err)
				assert.ErrorContains(t, err, fmt.Sprintf("%q node error occurred at", f.Nodes[0].Node.Addr()))
				assert.NotContains(t, err.Error(), fmt.Sprintf("%q node error occurred at", f.Nodes[1].Node.Addr()))
			},
		},
		{
			Name:    "PrimaryDeadOtherAlive",
			Fixture: newFixture(t, 2),
			Test: func(t *testing.T, f *fixture, o *nodeUpdateObserver, cl *Cluster) {
				f.Nodes[0].setStatus(nodeStatusStandby)
				waitForNode(t, o, cl.WaitForPrimary, nil)

				err := cl.Err()
				require.Error(t, err)
				assert.NotContains(t, err.Error(), fmt.Sprintf("%q node error occurred at", f.Nodes[0].Node.Addr()))
				assert.ErrorContains(t, err, fmt.Sprintf("%q node error occurred at", f.Nodes[1].Node.Addr()))
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.Name, func(t *testing.T) {
			defer input.Fixture.AssertExpectations(t)

			var o nodeUpdateObserver
			cl := setupCluster(t, input.Fixture, o.Tracer())
			defer func() { require.NoError(t, cl.Close()) }()

			input.Test(t, input.Fixture, &o, cl)
		})
	}
}

type nodeStatus int64

const (
	nodeStatusUnknown nodeStatus = iota
	nodeStatusPrimary
	nodeStatusStandby
)

type mockedNode struct {
	Node Node
	Mock sqlmock.Sqlmock
	st   int64
}

func (n *mockedNode) setStatus(s nodeStatus) {
	atomic.StoreInt64(&n.st, int64(s))
}

func (n *mockedNode) status() nodeStatus {
	return nodeStatus(atomic.LoadInt64(&n.st))
}

type nodeUpdateObserver struct {
	updatedNodes int64

	updatedNodesAtStart int64
	updatesObserved     int64
}

func (o *nodeUpdateObserver) StartObservation() {
	o.updatedNodesAtStart = atomic.LoadInt64(&o.updatedNodes)
	o.updatesObserved = 0
}

func (o *nodeUpdateObserver) ObservedUpdates() bool {
	updatedNodes := atomic.LoadInt64(&o.updatedNodes)

	// When we wait for a node, we are guaranteed to observe at least one update
	// only when two actually happened
	// TODO: its a mess, implement checker
	if updatedNodes-o.updatedNodesAtStart >= 2*(o.updatesObserved+1) {
		o.updatesObserved++
	}

	return o.updatesObserved >= 2
}

func (o *nodeUpdateObserver) Tracer() Tracer {
	return Tracer{
		UpdatedNodes: func(_ AliveNodes) { atomic.AddInt64(&o.updatedNodes, 1) },
	}
}

type fixture struct {
	TraceCounter *nodeUpdateObserver
	Nodes        []*mockedNode
}

func newFixture(t *testing.T, count int) *fixture {
	var f fixture
	for i := count; i > 0; i-- {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		require.NotNil(t, db)

		mock.ExpectClose()

		node := &mockedNode{
			Node: NewNode(uuid.Must(uuid.NewV4()).String(), db),
			Mock: mock,
			st:   int64(nodeStatusUnknown),
		}

		require.NotNil(t, node.Node)
		f.Nodes = append(f.Nodes, node)
	}

	require.Len(t, f.Nodes, count)

	return &f
}

func (f *fixture) ClusterNodes() []Node {
	var nodes []Node
	for _, node := range f.Nodes {
		nodes = append(nodes, node.Node)
	}

	return nodes
}

func (f *fixture) PrimaryChecker(_ context.Context, db *sql.DB) (bool, error) {
	for _, node := range f.Nodes {
		if node.Node.DB() == db {
			switch node.status() {
			case nodeStatusPrimary:
				return true, nil
			case nodeStatusStandby:
				return false, nil
			default:
				return false, errors.New("node is dead")
			}
		}
	}

	return false, errors.New("node not found in fixture")
}

func (f *fixture) AssertExpectations(t *testing.T) {
	for _, node := range f.Nodes {
		if node.Mock != nil { // We can use 'incomplete' fixture to test invalid cases
			assert.NoError(t, node.Mock.ExpectationsWereMet())
		}
	}
}
