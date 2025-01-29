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

package hasql_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// This example assumes you use pgx driver, but you can use anything that supports database/sql
	// _ "github.com/jackc/pgx/v4/stdlib"

	"golang.yandex/hasql"
	"golang.yandex/hasql/checkers"
)

func ExampleTracer() {
	const hostname = "host=host1.example.com"
	db, err := sql.Open("pgx", "host="+hostname)
	if err != nil {
		panic(err)
	}

	nodes := []hasql.Node{hasql.NewNode(hostname, db)}

	tracer := hasql.Tracer{
		UpdateNodes: func() {
			fmt.Println("Started updating nodes")
		},
		UpdatedNodes: func(nodes hasql.AliveNodes) {
			fmt.Printf("Finished updating nodes: %+v\n", nodes)
		},
		NodeDead: func(node hasql.Node, err error) {
			fmt.Printf("Node %q is dead: %s", node, err)
		},
		NodeAlive: func(node hasql.Node) {
			fmt.Printf("Node %q is alive", node)
		},
		NotifiedWaiters: func() {
			fmt.Println("Notified all waiters")
		},
	}

	c, err := hasql.NewCluster(nodes, checkers.PostgreSQL, hasql.WithTracer(tracer))
	if err != nil {
		panic(err)
	}
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = c.WaitForPrimary(ctx)
	if err != nil {
		panic(err)
	}
}
