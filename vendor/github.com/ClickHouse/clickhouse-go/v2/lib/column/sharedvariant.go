// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package column

import (
	"github.com/ClickHouse/ch-go/proto"
	"reflect"
)

type SharedVariant struct {
	name       string
	stringData String
}

func (c *SharedVariant) Name() string {
	return c.name
}

func (c *SharedVariant) Type() Type {
	return "SharedVariant"
}

func (c *SharedVariant) Rows() int {
	return c.stringData.Rows()
}

func (c *SharedVariant) Row(i int, ptr bool) any {
	return c.stringData.Row(i, ptr)
}

func (c *SharedVariant) ScanRow(dest any, row int) error {
	return c.stringData.ScanRow(dest, row)
}

func (c *SharedVariant) Append(v any) (nulls []uint8, err error) {
	return c.stringData.Append(v)
}

func (c *SharedVariant) AppendRow(v any) error {
	return c.stringData.AppendRow(v)
}

func (c *SharedVariant) Decode(reader *proto.Reader, rows int) error {
	return c.stringData.Decode(reader, rows)
}

func (c *SharedVariant) Encode(buffer *proto.Buffer) {
	c.stringData.Encode(buffer)
}

func (c *SharedVariant) ScanType() reflect.Type {
	return c.stringData.ScanType()
}

func (c *SharedVariant) Reset() {
	c.stringData.Reset()
}
