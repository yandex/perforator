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
	"sort"
	"sync"
	"time"
)

type checkedNode struct {
	Node    Node
	Latency time.Duration
}

type checkedNodesList []checkedNode

var _ sort.Interface = checkedNodesList{}

func (list checkedNodesList) Len() int {
	return len(list)
}

func (list checkedNodesList) Less(i, j int) bool {
	return list[i].Latency < list[j].Latency
}

func (list checkedNodesList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list checkedNodesList) Nodes() []Node {
	res := make([]Node, 0, len(list))
	for _, node := range list {
		res = append(res, node.Node)
	}

	return res
}

type groupedCheckedNodes struct {
	Primaries checkedNodesList
	Standbys  checkedNodesList
}

// Alive returns merged primaries and standbys sorted by latency. Primaries and standbys are expected to be
// sorted beforehand.
func (nodes groupedCheckedNodes) Alive() []Node {
	res := make([]Node, len(nodes.Primaries)+len(nodes.Standbys))

	var i int
	for len(nodes.Primaries) > 0 && len(nodes.Standbys) > 0 {
		if nodes.Primaries[0].Latency < nodes.Standbys[0].Latency {
			res[i] = nodes.Primaries[0].Node
			nodes.Primaries = nodes.Primaries[1:]
		} else {
			res[i] = nodes.Standbys[0].Node
			nodes.Standbys = nodes.Standbys[1:]
		}

		i++
	}

	for j := 0; j < len(nodes.Primaries); j++ {
		res[i] = nodes.Primaries[j].Node
		i++
	}

	for j := 0; j < len(nodes.Standbys); j++ {
		res[i] = nodes.Standbys[j].Node
		i++
	}

	return res
}

type checkExecutorFunc func(ctx context.Context, node Node) (bool, time.Duration, error)

// checkNodes takes slice of nodes, checks them in parallel and returns the alive ones.
// Accepts customizable executor which enables time-independent tests for node sorting based on 'latency'.
func checkNodes(ctx context.Context, nodes []Node, executor checkExecutorFunc, tracer Tracer, errCollector *errorsCollector) AliveNodes {
	checkedNodes := groupedCheckedNodes{
		Primaries: make(checkedNodesList, 0, len(nodes)),
		Standbys:  make(checkedNodesList, 0, len(nodes)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, node := range nodes {
		go func(node Node, wg *sync.WaitGroup) {
			defer wg.Done()

			primary, duration, err := executor(ctx, node)
			if err != nil {
				if tracer.NodeDead != nil {
					tracer.NodeDead(node, err)
				}
				if errCollector != nil {
					errCollector.Add(node.Addr(), err, time.Now())
				}
				return
			}
			if errCollector != nil {
				errCollector.Remove(node.Addr())
			}

			if tracer.NodeAlive != nil {
				tracer.NodeAlive(node)
			}

			nl := checkedNode{Node: node, Latency: duration}

			mu.Lock()
			defer mu.Unlock()
			if primary {
				checkedNodes.Primaries = append(checkedNodes.Primaries, nl)
			} else {
				checkedNodes.Standbys = append(checkedNodes.Standbys, nl)
			}
		}(node, &wg)
	}
	wg.Wait()

	sort.Sort(checkedNodes.Primaries)
	sort.Sort(checkedNodes.Standbys)

	return AliveNodes{
		Alive:     checkedNodes.Alive(),
		Primaries: checkedNodes.Primaries.Nodes(),
		Standbys:  checkedNodes.Standbys.Nodes(),
	}
}

// checkExecutor returns checkExecutorFunc which can execute supplied check.
func checkExecutor(checker NodeChecker) checkExecutorFunc {
	return func(ctx context.Context, node Node) (bool, time.Duration, error) {
		ts := time.Now()
		primary, err := checker(ctx, node.DB())
		d := time.Since(ts)
		if err != nil {
			return false, d, err
		}

		return primary, d, nil
	}
}
