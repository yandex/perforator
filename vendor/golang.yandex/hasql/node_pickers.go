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
	"math/rand"
	"sync/atomic"
)

// PickNodeRandom returns random node from nodes set
func PickNodeRandom() NodePicker {
	return func(nodes []Node) Node {
		return nodes[rand.Intn(len(nodes))]
	}
}

// PickNodeRoundRobin returns next node based on Round Robin algorithm
func PickNodeRoundRobin() NodePicker {
	var nodeIdx uint32
	return func(nodes []Node) Node {
		n := atomic.AddUint32(&nodeIdx, 1)
		return nodes[(int(n)-1)%len(nodes)]
	}
}

// PickNodeClosest returns node with least latency
func PickNodeClosest() NodePicker {
	return func(nodes []Node) Node {
		return nodes[0]
	}
}
