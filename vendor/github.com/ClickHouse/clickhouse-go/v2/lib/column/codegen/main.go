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

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"log"
	"os"
	"path"
	"sort"
	"text/template"
)

var (
	//go:embed column.tpl
	columnSrc string
	//go:embed array.tpl
	arraySrc string
)
var (
	types            []_type
	supportedGoTypes []string
)

type _type struct {
	Size   int
	ChType string
	GoType string
}

func init() {
	for _, size := range []int{8, 16, 32, 64} {
		types = append(types, _type{
			Size:   size,
			ChType: fmt.Sprintf("Int%d", size),
			GoType: fmt.Sprintf("int%d", size),
		}, _type{
			Size:   size,
			ChType: fmt.Sprintf("UInt%d", size),
			GoType: fmt.Sprintf("uint%d", size),
		})
	}
	for _, size := range []int{32, 64} {
		types = append(types, _type{
			Size:   size,
			ChType: fmt.Sprintf("Float%d", size),
			GoType: fmt.Sprintf("float%d", size),
		})
	}
	sort.Slice(types, func(i, j int) bool {
		return sequenceKey(types[i].ChType) < sequenceKey(types[j].ChType)
	})

	for _, typ := range types {
		supportedGoTypes = append(supportedGoTypes, typ.GoType)
	}
	supportedGoTypes = append(supportedGoTypes,
		"string", "[]byte", "sql.NullString",
		"int", "uint", "big.Int", "decimal.Decimal",
		"bool", "sql.NullBool",
		"time.Time", "sql.NullTime",
		"uuid.UUID",
		"netip.Addr", "net.IP", "proto.IPv6", "[16]byte",
		"orb.MultiPolygon", "orb.Point", "orb.Polygon", "orb.Ring",
	)
}
func write(name string, v any, t *template.Template) error {
	out := new(bytes.Buffer)
	if err := t.Execute(out, v); err != nil {
		return err
	}
	//	fmt.Println(out.String())
	data, err := format.Source(out.Bytes())
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(cwd, fmt.Sprintf("lib/column/%s.go", name)), data, 0o600); err != nil {
		return err
	}
	return nil
}

func main() {
	for name, tpl := range map[string]struct {
		template *template.Template
		args     any
	}{
		"column_gen": {template.Must(template.New("column").Parse(columnSrc)), types},
		"array_gen":  {template.Must(template.New("array").Parse(arraySrc)), supportedGoTypes},
	} {
		if err := write(name, tpl.args, tpl.template); err != nil {
			log.Fatal(err)
		}
	}
}

const maxByte = 1<<8 - 1

func isDigit(d byte) bool {
	return '0' <= d && d <= '9'
}

func sequenceKey(key string) string {
	sKey := make([]byte, 0, len(key)+8)
	j := -1
	for i := 0; i < len(key); i++ {
		b := key[i]
		if !isDigit(b) {
			sKey = append(sKey, b)
			j = -1
			continue
		}
		if j == -1 {
			sKey = append(sKey, 0x00)
			j = len(sKey) - 1
		}
		if sKey[j] == 1 && sKey[j+1] == '0' {
			sKey[j+1] = b
			continue
		}
		if sKey[j]+1 > maxByte {
			panic("sequenceKey: invalid key")
		}
		sKey = append(sKey, b)
		sKey[j]++
	}
	return string(sKey)
}
