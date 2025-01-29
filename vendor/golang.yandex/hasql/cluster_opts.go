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

import "time"

// ClusterOption is a functional option type for Cluster constructor
type ClusterOption func(*Cluster)

// WithUpdateInterval sets interval between cluster node updates
func WithUpdateInterval(d time.Duration) ClusterOption {
	return func(cl *Cluster) {
		cl.updateInterval = d
	}
}

// WithUpdateTimeout sets ping timeout for update of each node in cluster
func WithUpdateTimeout(d time.Duration) ClusterOption {
	return func(cl *Cluster) {
		cl.updateTimeout = d
	}
}

// WithNodePicker sets algorithm for node selection (e.g. random, round robin etc)
func WithNodePicker(picker NodePicker) ClusterOption {
	return func(cl *Cluster) {
		cl.picker = picker
	}
}

// WithTracer sets tracer for actions happening in the background
func WithTracer(tracer Tracer) ClusterOption {
	return func(cl *Cluster) {
		cl.tracer = tracer
	}
}
