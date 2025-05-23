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
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
)

const JSONObjectSerializationVersion uint64 = 0
const JSONStringSerializationVersion uint64 = 1
const JSONUnsetSerializationVersion uint64 = math.MaxUint64
const DefaultMaxDynamicPaths = 1024

type JSON struct {
	chType Type
	tz     *time.Location
	name   string
	rows   int

	serializationVersion uint64

	jsonStrings String

	typedPaths      []string
	typedPathsIndex map[string]int
	typedColumns    []Interface

	skipPaths      []string
	skipPathsIndex map[string]int

	dynamicPaths      []string
	dynamicPathsIndex map[string]int
	dynamicColumns    []*Dynamic

	maxDynamicPaths   int
	maxDynamicTypes   int
	totalDynamicPaths int
}

func (c *JSON) parse(t Type, tz *time.Location) (_ *JSON, err error) {
	c.chType = t
	c.tz = tz
	tStr := string(t)

	c.serializationVersion = JSONUnsetSerializationVersion
	c.typedPathsIndex = make(map[string]int)
	c.skipPathsIndex = make(map[string]int)
	c.dynamicPathsIndex = make(map[string]int)
	c.maxDynamicPaths = DefaultMaxDynamicPaths
	c.maxDynamicTypes = DefaultMaxDynamicTypes

	if tStr == "JSON" {
		return c, nil
	}

	if !strings.HasPrefix(tStr, "JSON(") || !strings.HasSuffix(tStr, ")") {
		return nil, &UnsupportedColumnTypeError{t: t}
	}

	typePartsStr := strings.TrimPrefix(tStr, "JSON(")
	typePartsStr = strings.TrimSuffix(typePartsStr, ")")

	typeParts := splitWithDelimiters(typePartsStr)
	for _, typePart := range typeParts {
		typePart = strings.TrimSpace(typePart)

		if strings.HasPrefix(typePart, "max_dynamic_paths=") {
			v := strings.TrimPrefix(typePart, "max_dynamic_paths=")
			if maxPaths, err := strconv.Atoi(v); err == nil {
				c.maxDynamicPaths = maxPaths
			}

			continue
		}

		if strings.HasPrefix(typePart, "max_dynamic_types=") {
			v := strings.TrimPrefix(typePart, "max_dynamic_types=")
			if maxTypes, err := strconv.Atoi(v); err == nil {
				c.maxDynamicTypes = maxTypes
			}

			continue
		}

		if strings.HasPrefix(typePart, "SKIP REGEXP") {
			pattern := strings.TrimPrefix(typePart, "SKIP REGEXP")
			pattern = strings.Trim(pattern, " '")
			c.skipPaths = append(c.skipPaths, pattern)
			c.skipPathsIndex[pattern] = len(c.skipPaths) - 1

			continue
		}

		if strings.HasPrefix(typePart, "SKIP") {
			path := strings.TrimPrefix(typePart, "SKIP")
			path = strings.Trim(path, " `")
			c.skipPaths = append(c.skipPaths, path)
			c.skipPathsIndex[path] = len(c.skipPaths) - 1

			continue
		}

		typedPathParts := strings.SplitN(typePart, " ", 2)
		if len(typedPathParts) != 2 {
			continue
		}

		typedPath := strings.Trim(typedPathParts[0], "`")
		typeName := strings.TrimSpace(typedPathParts[1])

		c.typedPaths = append(c.typedPaths, typedPath)
		c.typedPathsIndex[typedPath] = len(c.typedPaths) - 1

		col, err := Type(typeName).Column("", tz)
		if err != nil {
			return nil, fmt.Errorf("failed to init column of type \"%s\" at path \"%s\": %w", typeName, typedPath, err)
		}

		c.typedColumns = append(c.typedColumns, col)
	}

	return c, nil
}

func (c *JSON) hasTypedPath(path string) bool {
	_, ok := c.typedPathsIndex[path]
	return ok
}

func (c *JSON) hasDynamicPath(path string) bool {
	_, ok := c.dynamicPathsIndex[path]
	return ok
}

func (c *JSON) hasSkipPath(path string) bool {
	_, ok := c.skipPathsIndex[path]
	return ok
}

// pathHasNestedValues returns true if the provided path has child paths in typed or dynamic paths
func (c *JSON) pathHasNestedValues(path string) bool {
	for _, typedPath := range c.typedPaths {
		if strings.HasPrefix(typedPath, path+".") {
			return true
		}
	}

	for _, dynamicPath := range c.dynamicPaths {
		if strings.HasPrefix(dynamicPath, path+".") {
			return true
		}
	}

	return false
}

// valueAtPath returns the row value at the specified path, typed or dynamic
func (c *JSON) valueAtPath(path string, row int, ptr bool) any {
	if colIndex, ok := c.typedPathsIndex[path]; ok {
		return c.typedColumns[colIndex].Row(row, ptr)
	}

	if colIndex, ok := c.dynamicPathsIndex[path]; ok {
		return c.dynamicColumns[colIndex].Row(row, ptr)
	}

	return nil
}

// scanTypedPathToValue scans the provided typed path into a `reflect.Value`
func (c *JSON) scanTypedPathToValue(path string, row int, value reflect.Value) error {
	colIndex, ok := c.typedPathsIndex[path]
	if !ok {
		return fmt.Errorf("typed path \"%s\" does not exist in JSON column", path)
	}

	col := c.typedColumns[colIndex]
	err := col.ScanRow(value.Addr().Interface(), row)
	if err != nil {
		return fmt.Errorf("failed to scan %s column into typed path \"%s\": %w", col.Type(), path, err)
	}

	return nil
}

// scanDynamicPathToValue scans the provided typed path into a `reflect.Value`
func (c *JSON) scanDynamicPathToValue(path string, row int, value reflect.Value) error {
	colIndex, ok := c.dynamicPathsIndex[path]
	if !ok {
		return fmt.Errorf("dynamic path \"%s\" does not exist in JSON column", path)
	}

	col := c.dynamicColumns[colIndex]
	err := col.ScanRow(value.Addr().Interface(), row)
	if err != nil {
		return fmt.Errorf("failed to scan %s column into dynamic path \"%s\": %w", col.Type(), path, err)
	}

	return nil
}

func (c *JSON) rowAsJSON(row int) *chcol.JSON {
	obj := chcol.NewJSON()

	for i, path := range c.typedPaths {
		col := c.typedColumns[i]
		obj.SetValueAtPath(path, col.Row(row, false))
	}

	for i, path := range c.dynamicPaths {
		col := c.dynamicColumns[i]
		obj.SetValueAtPath(path, col.Row(row, false))
	}

	return obj
}

func (c *JSON) Name() string {
	return c.name
}

func (c *JSON) Type() Type {
	return c.chType
}

func (c *JSON) Rows() int {
	return c.rows
}

func (c *JSON) Row(row int, ptr bool) any {
	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		return c.rowAsJSON(row)
	case JSONStringSerializationVersion:
		return c.jsonStrings.Row(row, ptr)
	default:
		return nil
	}
}

func (c *JSON) ScanRow(dest any, row int) error {
	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		return c.scanRowObject(dest, row)
	case JSONStringSerializationVersion:
		return c.scanRowString(dest, row)
	default:
		return fmt.Errorf("unsupported JSON serialization version for scan: %d", c.serializationVersion)
	}
}

func (c *JSON) scanRowObject(dest any, row int) error {
	switch v := dest.(type) {
	case *chcol.JSON:
		obj := c.rowAsJSON(row)
		*v = *obj
		return nil
	case **chcol.JSON:
		obj := c.rowAsJSON(row)
		**v = *obj
		return nil
	case chcol.JSONDeserializer:
		obj := c.rowAsJSON(row)
		err := v.DeserializeClickHouseJSON(obj)
		if err != nil {
			return fmt.Errorf("failed to deserialize using DeserializeClickHouseJSON: %w", err)
		}

		return nil
	}

	switch val := reflect.ValueOf(dest); val.Kind() {
	case reflect.Pointer:
		if val.Elem().Kind() == reflect.Struct {
			return c.scanIntoStruct(dest, row)
		} else if val.Elem().Kind() == reflect.Map {
			return c.scanIntoMap(dest, row)
		}
	}

	return fmt.Errorf("destination must be a pointer to struct or map, or %s. hint: enable \"output_format_native_write_json_as_string\" setting for string decoding", scanTypeJSON.String())
}

func (c *JSON) scanRowString(dest any, row int) error {
	return c.jsonStrings.ScanRow(dest, row)
}

func (c *JSON) Append(v any) (nulls []uint8, err error) {
	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		return c.appendObject(v)
	case JSONStringSerializationVersion:
		return c.appendString(v)
	default:
		// Unset serialization preference, try string first unless its specifically JSON
		switch v.(type) {
		case []chcol.JSON:
			c.serializationVersion = JSONObjectSerializationVersion
			return c.appendObject(v)
		case []*chcol.JSON:
			c.serializationVersion = JSONObjectSerializationVersion
			return c.appendObject(v)
		case []chcol.JSONSerializer:
			c.serializationVersion = JSONObjectSerializationVersion
			return c.appendObject(v)
		}

		var err error
		if _, err = c.appendString(v); err == nil {
			c.serializationVersion = JSONStringSerializationVersion
			return nil, nil
		} else if _, err = c.appendObject(v); err == nil {
			c.serializationVersion = JSONObjectSerializationVersion
			return nil, nil
		}

		return nil, fmt.Errorf("unsupported type \"%s\" for JSON column, must use slice of string, []byte, struct, map, or *%s: %w", reflect.TypeOf(v).String(), scanTypeJSON.String(), err)
	}
}

func (c *JSON) appendObject(v any) (nulls []uint8, err error) {
	switch vv := v.(type) {
	case []chcol.JSON:
		for i, obj := range vv {
			err := c.AppendRow(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	case []*chcol.JSON:
		for i, obj := range vv {
			err := c.AppendRow(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	case []chcol.JSONSerializer:
		for i, obj := range vv {
			err := c.AppendRow(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	}

	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Slice {
		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(c.chType),
			From: fmt.Sprintf("%T", v),
			Hint: "value must be a slice",
		}
	}
	for i := 0; i < value.Len(); i++ {
		if err := c.AppendRow(value.Index(i)); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (c *JSON) appendString(v any) (nulls []uint8, err error) {
	nulls, err = c.jsonStrings.Append(v)
	if err != nil {
		return nil, err
	}

	c.rows = c.jsonStrings.Rows()
	return nulls, nil
}

func (c *JSON) AppendRow(v any) error {
	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		return c.appendRowObject(v)
	case JSONStringSerializationVersion:
		return c.appendRowString(v)
	default:
		// Unset serialization preference, try string first unless its specifically JSON
		switch v.(type) {
		case chcol.JSON:
			c.serializationVersion = JSONObjectSerializationVersion
			return c.appendRowObject(v)
		case *chcol.JSON:
			c.serializationVersion = JSONObjectSerializationVersion
			return c.appendRowObject(v)
		case chcol.JSONSerializer:
			c.serializationVersion = JSONObjectSerializationVersion
			return c.appendRowObject(v)
		}

		var err error
		if err = c.appendRowString(v); err == nil {
			c.serializationVersion = JSONStringSerializationVersion
			return nil
		} else if err = c.appendRowObject(v); err == nil {
			c.serializationVersion = JSONObjectSerializationVersion
			return nil
		}

		return fmt.Errorf("unsupported type \"%s\" for JSON column, must use string, []byte, *struct, map, or *clickhouse.JSON: %w", reflect.TypeOf(v).String(), err)
	}
}

func (c *JSON) appendRowObject(v any) error {
	var obj *chcol.JSON
	switch vv := v.(type) {
	case chcol.JSON:
		obj = &vv
	case *chcol.JSON:
		obj = vv
	case chcol.JSONSerializer:
		var err error
		obj, err = vv.SerializeClickHouseJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize using SerializeClickHouseJSON: %w", err)
		}
	}

	if obj == nil && v != nil {
		var err error
		switch val := reflect.ValueOf(v); val.Kind() {
		case reflect.Pointer:
			if val.Elem().Kind() == reflect.Struct {
				obj, err = structToJSON(v)
			} else if val.Elem().Kind() == reflect.Map {
				obj, err = mapToJSON(v)
			}
		case reflect.Struct:
			obj, err = structToJSON(v)
		case reflect.Map:
			obj, err = mapToJSON(v)
		}

		if err != nil {
			return fmt.Errorf("failed to convert value to JSON: %w", err)
		}
	}

	if obj == nil {
		obj = chcol.NewJSON()
	}
	valuesByPath := obj.ValuesByPath()

	// Match typed paths first
	for i, typedPath := range c.typedPaths {
		// Even if value is nil, we must append a value for this row.
		// nil is a valid value for most column types, with most implementations putting a zero value.
		// If the column doesn't support appending nil, then the user must provide a zero value.
		value, _ := valuesByPath[typedPath]

		col := c.typedColumns[i]
		err := col.AppendRow(value)
		if err != nil {
			return fmt.Errorf("failed to append type %s to json column at typed path %s: %w", col.Type(), typedPath, err)
		}
	}

	// Verify all dynamic paths have an equal number of rows by appending nil for all unspecified dynamic paths
	for _, dynamicPath := range c.dynamicPaths {
		if _, ok := valuesByPath[dynamicPath]; !ok {
			valuesByPath[dynamicPath] = nil
		}
	}

	// Match or add dynamic paths
	for objPath, value := range valuesByPath {
		if c.hasTypedPath(objPath) || c.hasSkipPath(objPath) {
			continue
		}

		if dynamicPathIndex, ok := c.dynamicPathsIndex[objPath]; ok {
			err := c.dynamicColumns[dynamicPathIndex].AppendRow(value)
			if err != nil {
				return fmt.Errorf("failed to append to json column at dynamic path \"%s\": %w", objPath, err)
			}
		} else {
			// Path doesn't exist, add new dynamic path + column
			parsedColDynamic, _ := Type("Dynamic").Column("", c.tz)
			colDynamic := parsedColDynamic.(*Dynamic)

			// New path must back-fill nils for each row
			for i := 0; i < c.rows; i++ {
				err := colDynamic.AppendRow(nil)
				if err != nil {
					return fmt.Errorf("failed to back-fill json column at new dynamic path \"%s\" index %d: %w", objPath, i, err)
				}
			}

			err := colDynamic.AppendRow(value)
			if err != nil {
				return fmt.Errorf("failed to append to json column at new dynamic path \"%s\": %w", objPath, err)
			}

			c.dynamicPaths = append(c.dynamicPaths, objPath)
			c.dynamicPathsIndex[objPath] = len(c.dynamicPaths) - 1
			c.dynamicColumns = append(c.dynamicColumns, colDynamic)
			c.totalDynamicPaths++
		}
	}

	c.rows++
	return nil
}

func (c *JSON) appendRowString(v any) error {
	err := c.jsonStrings.AppendRow(v)
	if err != nil {
		return err
	}

	c.rows++
	return nil
}

func (c *JSON) encodeObjectHeader(buffer *proto.Buffer) {
	buffer.PutUVarInt(uint64(c.maxDynamicPaths))
	buffer.PutUVarInt(uint64(c.totalDynamicPaths))

	for _, dynamicPath := range c.dynamicPaths {
		buffer.PutString(dynamicPath)
	}

	for _, col := range c.dynamicColumns {
		col.encodeHeader(buffer)
	}
}

func (c *JSON) encodeObjectData(buffer *proto.Buffer) {
	for _, col := range c.typedColumns {
		col.Encode(buffer)
	}

	for _, col := range c.dynamicColumns {
		col.encodeData(buffer)
	}

	// SharedData per row, empty for now.
	for i := 0; i < c.rows; i++ {
		buffer.PutUInt64(0)
	}
}

func (c *JSON) encodeStringData(buffer *proto.Buffer) {
	c.jsonStrings.Encode(buffer)
}

func (c *JSON) WriteStatePrefix(buffer *proto.Buffer) error {
	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		buffer.PutUInt64(JSONObjectSerializationVersion)
		c.encodeObjectHeader(buffer)

		return nil
	case JSONStringSerializationVersion:
		buffer.PutUInt64(JSONStringSerializationVersion)

		return nil
	default:
		// If the column is an array, it can be empty but still require a prefix.
		// Use string encoding since it's smaller
		buffer.PutUInt64(JSONStringSerializationVersion)

		return nil
	}
}

func (c *JSON) Encode(buffer *proto.Buffer) {
	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		c.encodeObjectData(buffer)
		return
	case JSONStringSerializationVersion:
		c.encodeStringData(buffer)
		return
	}
}

func (c *JSON) ScanType() reflect.Type {
	return scanTypeJSON
}

func (c *JSON) Reset() {
	c.rows = 0

	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		for _, col := range c.typedColumns {
			col.Reset()
		}

		for _, col := range c.dynamicColumns {
			col.Reset()
		}

		return
	case JSONStringSerializationVersion:
		c.jsonStrings.Reset()
		return
	}
}

func (c *JSON) decodeObjectHeader(reader *proto.Reader) error {
	maxDynamicPaths, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read max dynamic paths for json column: %w", err)
	}
	c.maxDynamicPaths = int(maxDynamicPaths)

	totalDynamicPaths, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read total dynamic paths for json column: %w", err)
	}
	c.totalDynamicPaths = int(totalDynamicPaths)

	c.dynamicPaths = make([]string, 0, totalDynamicPaths)
	for i := 0; i < int(totalDynamicPaths); i++ {
		dynamicPath, err := reader.Str()
		if err != nil {
			return fmt.Errorf("failed to read dynamic path name bytes at index %d for json column: %w", i, err)
		}

		c.dynamicPaths = append(c.dynamicPaths, dynamicPath)
		c.dynamicPathsIndex[dynamicPath] = len(c.dynamicPaths) - 1
	}

	c.dynamicColumns = make([]*Dynamic, 0, totalDynamicPaths)
	for _, dynamicPath := range c.dynamicPaths {
		parsedColDynamic, _ := Type("Dynamic").Column("", c.tz)
		colDynamic := parsedColDynamic.(*Dynamic)

		err := colDynamic.decodeHeader(reader)
		if err != nil {
			return fmt.Errorf("failed to decode dynamic header at path %s for json column: %w", dynamicPath, err)
		}

		c.dynamicColumns = append(c.dynamicColumns, colDynamic)
	}

	return nil
}

func (c *JSON) decodeObjectData(reader *proto.Reader, rows int) error {
	for i, col := range c.typedColumns {
		typedPath := c.typedPaths[i]

		err := col.Decode(reader, rows)
		if err != nil {
			return fmt.Errorf("failed to decode %s typed path %s for json column: %w", col.Type(), typedPath, err)
		}
	}

	for i, col := range c.dynamicColumns {
		dynamicPath := c.dynamicPaths[i]

		err := col.decodeData(reader, rows)
		if err != nil {
			return fmt.Errorf("failed to decode dynamic path %s for json column: %w", dynamicPath, err)
		}
	}

	// SharedData per row, ignored for now. May cause stream offset issues if present
	_, err := reader.ReadRaw(8 * rows) // one UInt64 per row
	if err != nil {
		return fmt.Errorf("failed to read shared data for json column: %w", err)
	}

	return nil
}

func (c *JSON) decodeStringData(reader *proto.Reader, rows int) error {
	return c.jsonStrings.Decode(reader, rows)
}

func (c *JSON) ReadStatePrefix(reader *proto.Reader) error {
	jsonSerializationVersion, err := reader.UInt64()
	if err != nil {
		return fmt.Errorf("failed to read json serialization version: %w", err)
	}

	c.serializationVersion = jsonSerializationVersion

	switch jsonSerializationVersion {
	case JSONObjectSerializationVersion:
		err := c.decodeObjectHeader(reader)
		if err != nil {
			return fmt.Errorf("failed to decode json object header: %w", err)
		}

		return nil
	case JSONStringSerializationVersion:
		return nil
	default:
		return fmt.Errorf("unsupported JSON serialization version for prefix decode: %d", jsonSerializationVersion)
	}
}

func (c *JSON) Decode(reader *proto.Reader, rows int) error {
	c.rows = rows

	switch c.serializationVersion {
	case JSONObjectSerializationVersion:
		err := c.decodeObjectData(reader, rows)
		if err != nil {
			return fmt.Errorf("failed to decode json object data: %w", err)
		}

		return nil
	case JSONStringSerializationVersion:
		err := c.decodeStringData(reader, rows)
		if err != nil {
			return fmt.Errorf("failed to decode json string data: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("unsupported JSON serialization version for decode: %d", c.serializationVersion)
	}
}

// splitWithDelimiters splits the string while considering backticks and parentheses
func splitWithDelimiters(s string) []string {
	var parts []string
	var currentPart strings.Builder
	var brackets int
	inBackticks := false

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '`':
			inBackticks = !inBackticks
			currentPart.WriteByte(s[i])
		case '(':
			brackets++
			currentPart.WriteByte(s[i])
		case ')':
			brackets--
			currentPart.WriteByte(s[i])
		case ',':
			if !inBackticks && brackets == 0 {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			} else {
				currentPart.WriteByte(s[i])
			}
		default:
			currentPart.WriteByte(s[i])
		}
	}

	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	return parts
}
