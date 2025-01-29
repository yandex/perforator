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
	"database/sql"

	"github.com/jmoiron/sqlx"

	"golang.yandex/hasql"
)

// Node of single cluster
type Node interface {
	hasql.Node

	DBx() *sqlx.DB
}

type sqlxNode struct {
	addr string
	dbx  *sqlx.DB
}

var _ Node = &sqlxNode{}

// NewNode constructs node from sqlx.DB
func NewNode(addr string, db *sqlx.DB) Node {
	return &sqlxNode{
		addr: addr,
		dbx:  db,
	}
}

// Addr returns node's address
func (n *sqlxNode) Addr() string {
	return n.addr
}

// DB returns node's database/sql DB
func (n *sqlxNode) DB() *sql.DB {
	return n.dbx.DB
}

// DBx returns node's sqlx.DB
func (n *sqlxNode) DBx() *sqlx.DB {
	return n.dbx
}

// String implements Stringer
func (n *sqlxNode) String() string {
	return n.addr
}

func uncheckedSQLxNode(node hasql.Node) Node {
	if node == nil {
		return nil
	}

	return node.(*sqlxNode)
}

func checkedSQLxNode(node hasql.Node, err error) (Node, error) {
	if err != nil {
		return nil, err
	}

	return uncheckedSQLxNode(node), nil
}
