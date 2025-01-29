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

	"golang.yandex/hasql"
)

// Cluster consists of number of 'nodes' of a single SQL database.
// Background goroutine periodically checks nodes and updates their status.
type Cluster struct {
	*hasql.Cluster
}

// NewCluster constructs cluster object representing a single 'cluster' of SQL database.
// Close function must be called when cluster is not needed anymore.
func NewCluster(nodes []Node, checker NodeChecker, opts ...ClusterOption) (*Cluster, error) {
	sqlNodes := make([]hasql.Node, 0, len(nodes))
	for _, n := range nodes {
		sqlNodes = append(sqlNodes, n)
	}

	cl, err := hasql.NewCluster(sqlNodes, checker, opts...)
	if err != nil {
		return nil, err
	}

	return &Cluster{Cluster: cl}, nil
}

// WaitForAlive node to appear or until context is canceled
func (cl *Cluster) WaitForAlive(ctx context.Context) (Node, error) {
	return checkedSQLxNode(cl.Cluster.WaitForAlive(ctx))
}

// WaitForPrimary node to appear or until context is canceled
func (cl *Cluster) WaitForPrimary(ctx context.Context) (Node, error) {
	return checkedSQLxNode(cl.Cluster.WaitForPrimary(ctx))
}

// WaitForStandby node to appear or until context is canceled
func (cl *Cluster) WaitForStandby(ctx context.Context) (Node, error) {
	return checkedSQLxNode(cl.Cluster.WaitForStandby(ctx))
}

// WaitForPrimaryPreferred node to appear or until context is canceled
func (cl *Cluster) WaitForPrimaryPreferred(ctx context.Context) (Node, error) {
	return checkedSQLxNode(cl.Cluster.WaitForPrimaryPreferred(ctx))
}

// WaitForStandbyPreferred node to appear or until context is canceled
func (cl *Cluster) WaitForStandbyPreferred(ctx context.Context) (Node, error) {
	return checkedSQLxNode(cl.Cluster.WaitForStandbyPreferred(ctx))
}

// WaitForNode with specified status to appear or until context is canceled
func (cl *Cluster) WaitForNode(ctx context.Context, criteria NodeStateCriteria) (Node, error) {
	return checkedSQLxNode(cl.Cluster.WaitForNode(ctx, criteria))
}

// Alive returns node that is considered alive
func (cl *Cluster) Alive() Node {
	return uncheckedSQLxNode(cl.Cluster.Alive())
}

// Primary returns first available node that is considered alive and is primary (able to execute write operations)
func (cl *Cluster) Primary() Node {
	return uncheckedSQLxNode(cl.Cluster.Primary())
}

// Standby returns node that is considered alive and is standby (unable to execute write operations)
func (cl *Cluster) Standby() Node {
	return uncheckedSQLxNode(cl.Cluster.Standby())
}

// PrimaryPreferred returns primary node if possible, standby otherwise
func (cl *Cluster) PrimaryPreferred() Node {
	return uncheckedSQLxNode(cl.Cluster.PrimaryPreferred())
}

// StandbyPreferred returns standby node if possible, primary otherwise
func (cl *Cluster) StandbyPreferred() Node {
	return uncheckedSQLxNode(cl.Cluster.StandbyPreferred())
}

// Node returns cluster node with specified status.
func (cl *Cluster) Node(criteria NodeStateCriteria) Node {
	return uncheckedSQLxNode(cl.Cluster.Node(criteria))
}
