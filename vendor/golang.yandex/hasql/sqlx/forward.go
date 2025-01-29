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

import "golang.yandex/hasql"

type (
	// ClusterOption is a functional option type for Cluster constructor
	ClusterOption = hasql.ClusterOption
	// NodeStateCriteria for choosing a node
	NodeStateCriteria = hasql.NodeStateCriteria
	// NodeChecker is a signature for functions that check if specific node is alive and is primary.
	// Returns true for primary and false if not. If error is returned, node is considered dead.
	// Check function can be used to perform a query returning single boolean value that signals
	// if node is primary or not.
	NodeChecker = hasql.NodeChecker
	// NodePicker is a signature for functions that determine how to pick single node from set of nodes.
	// Nodes passed to the picker function are sorted according to latency (from lowest to greatest).
	NodePicker = hasql.NodePicker
	// AliveNodes of Cluster
	AliveNodes = hasql.AliveNodes
	// Tracer is a set of hooks to run at various stages of background nodes status update.
	// Any particular hook may be nil. Functions may be called concurrently from different goroutines.
	Tracer = hasql.Tracer
)

var (
	// Alive for choosing any alive node
	Alive = hasql.Alive
	// Primary for choosing primary node
	Primary = hasql.Primary
	// Standby for choosing standby node
	Standby = hasql.Standby
	// PreferPrimary for choosing primary or any alive node
	PreferPrimary = hasql.PreferPrimary
	// PreferStandby for choosing standby or any alive node
	PreferStandby = hasql.PreferStandby

	// WithUpdateInterval sets interval between cluster node updates
	WithUpdateInterval = hasql.WithUpdateInterval
	// WithUpdateTimeout sets ping timeout for update of each node in cluster
	WithUpdateTimeout = hasql.WithUpdateTimeout
	// WithNodePicker sets algorithm for node selection (e.g. random, round robin etc)
	WithNodePicker = hasql.WithNodePicker
	// WithTracer sets tracer for actions happening in the background
	WithTracer = hasql.WithTracer

	// PickNodeRandom returns random node from nodes set
	PickNodeRandom = hasql.PickNodeRandom
	// PickNodeRoundRobin returns next node based on Round Robin algorithm
	PickNodeRoundRobin = hasql.PickNodeRoundRobin
	// PickNodeClosest returns node with least latency
	PickNodeClosest = hasql.PickNodeClosest
)
