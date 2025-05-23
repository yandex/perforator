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
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
)

const SupportedDynamicSerializationVersion = 1
const DefaultMaxDynamicTypes = 32

type Dynamic struct {
	chType Type
	tz     *time.Location

	name string

	maxTypes       uint8
	totalTypes     uint8
	typeNames      []string
	typeNamesIndex map[string]int

	variant Variant
}

func (c *Dynamic) parse(t Type, tz *time.Location) (_ *Dynamic, err error) {
	c.chType = t
	c.tz = tz
	tStr := string(t)

	// SharedVariant is special, and does not count against totalTypes
	c.typeNamesIndex = make(map[string]int)
	c.variant.columnTypeIndex = make(map[string]uint8)
	sv, _ := Type("SharedVariant").Column("", tz)
	c.addColumn(sv)

	c.maxTypes = DefaultMaxDynamicTypes
	c.totalTypes = 0 // Reset to 0 after adding SharedVariant

	if tStr == "Dynamic" {
		return c, nil
	}

	if !strings.HasPrefix(tStr, "Dynamic(") || !strings.HasSuffix(tStr, ")") {
		return nil, &UnsupportedColumnTypeError{t: t}
	}

	typeParamsStr := strings.TrimPrefix(tStr, "Dynamic(")
	typeParamsStr = strings.TrimSuffix(typeParamsStr, ")")

	if strings.HasPrefix(typeParamsStr, "max_types=") {
		v := strings.TrimPrefix(typeParamsStr, "max_types=")
		if maxTypes, err := strconv.Atoi(v); err == nil {
			c.maxTypes = uint8(maxTypes)
		}
	}

	return c, nil
}

func (c *Dynamic) addColumn(col Interface) {
	typeName := string(col.Type())
	c.typeNames = append(c.typeNames, typeName)
	c.typeNamesIndex[typeName] = len(c.typeNames) - 1
	c.totalTypes++
	c.variant.addColumn(col)
}

func (c *Dynamic) Name() string {
	return c.name
}

func (c *Dynamic) Type() Type {
	return c.chType
}

func (c *Dynamic) Rows() int {
	return c.variant.Rows()
}

func (c *Dynamic) Row(i int, ptr bool) any {
	typeIndex := c.variant.discriminators[i]
	offsetIndex := c.variant.offsets[i]
	var value any
	var chType string
	if typeIndex != NullVariantDiscriminator {
		value = c.variant.columns[typeIndex].Row(offsetIndex, ptr)
		chType = string(c.variant.columns[typeIndex].Type())
	}

	dyn := chcol.NewDynamicWithType(value, chType)
	if ptr {
		return &dyn
	}

	return dyn
}

func (c *Dynamic) ScanRow(dest any, row int) error {
	typeIndex := c.variant.discriminators[row]
	offsetIndex := c.variant.offsets[row]
	var value any
	var chType string
	if typeIndex != NullVariantDiscriminator {
		value = c.variant.columns[typeIndex].Row(offsetIndex, false)
		chType = string(c.variant.columns[typeIndex].Type())
	}

	switch v := dest.(type) {
	case *chcol.Dynamic:
		dyn := chcol.NewDynamicWithType(value, chType)
		*v = dyn
	case **chcol.Dynamic:
		dyn := chcol.NewDynamicWithType(value, chType)
		**v = dyn
	default:
		if typeIndex == NullVariantDiscriminator {
			return nil
		}

		if err := c.variant.columns[typeIndex].ScanRow(dest, offsetIndex); err != nil {
			return err
		}
	}

	return nil
}

func (c *Dynamic) Append(v any) (nulls []uint8, err error) {
	switch vv := v.(type) {
	case []chcol.Dynamic:
		for i, dyn := range vv {
			err := c.AppendRow(dyn)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	case []*chcol.Dynamic:
		for i, dyn := range vv {
			err := c.AppendRow(dyn)
			if err != nil {
				return nil, fmt.Errorf("failed to AppendRow at index %d: %w", i, err)
			}
		}

		return nil, nil
	default:
		if valuer, ok := v.(driver.Valuer); ok {
			val, err := valuer.Value()
			if err != nil {
				return nil, &ColumnConverterError{
					Op:   "Append",
					To:   string(c.chType),
					From: fmt.Sprintf("%T", v),
					Hint: "could not get driver.Valuer value",
				}
			}

			return c.Append(val)
		}

		return nil, &ColumnConverterError{
			Op:   "Append",
			To:   string(c.chType),
			From: fmt.Sprintf("%T", v),
		}
	}
}

func (c *Dynamic) AppendRow(v any) error {
	var requestedType string
	switch vv := v.(type) {
	case nil:
		c.variant.appendNullRow()
		return nil
	case chcol.Dynamic:
		requestedType = vv.Type()
		v = vv.Any()
		if vv.Nil() {
			c.variant.appendNullRow()
			return nil
		}
	case *chcol.Dynamic:
		requestedType = vv.Type()
		v = vv.Any()
		if vv.Nil() {
			c.variant.appendNullRow()
			return nil
		}
	}

	if requestedType != "" {
		var col Interface
		colIndex, ok := c.typeNamesIndex[requestedType]
		if ok {
			col = c.variant.columns[colIndex]
		} else {
			newCol, err := Type(requestedType).Column("", c.tz)
			if err != nil {
				return fmt.Errorf("value \"%v\" cannot be stored in dynamic column %s with requested type %s: unable to append type: %w", v, c.chType, requestedType, err)
			}

			c.addColumn(newCol)
			colIndex = int(c.totalTypes)
			col = newCol
		}

		if err := col.AppendRow(v); err != nil {
			return fmt.Errorf("value \"%v\" cannot be stored in dynamic column %s with requested type %s: %w", v, c.chType, requestedType, err)
		}

		c.variant.appendDiscriminatorRow(uint8(colIndex))
		return nil
	}

	// If preferred type wasn't provided, try each column
	for i, col := range c.variant.columns {
		if c.typeNames[i] == "SharedVariant" {
			// Do not try to fit into SharedVariant
			continue
		}

		if err := col.AppendRow(v); err == nil {
			c.variant.appendDiscriminatorRow(uint8(i))
			return nil
		}
	}

	// If no existing columns match, try matching a ClickHouse type from common Go types
	inferredTypeName := inferClickHouseTypeFromGoType(v)
	if inferredTypeName != "" {
		return c.AppendRow(chcol.NewDynamicWithType(v, inferredTypeName))
	}

	return fmt.Errorf("value \"%v\" cannot be stored in dynamic column: no compatible types. hint: use clickhouse.DynamicWithType to wrap the value", v)
}

func (c *Dynamic) sortColumnsForEncoding() {
	previousTypeNames := make([]string, 0, len(c.typeNames))
	previousTypeNames = append(previousTypeNames, c.typeNames...)
	slices.Sort(c.typeNames)

	for i, typeName := range c.typeNames {
		c.typeNamesIndex[typeName] = i
		c.variant.columnTypeIndex[typeName] = uint8(i)
	}

	sortedDiscriminatorMap := make([]uint8, len(c.variant.columns))
	sortedColumns := make([]Interface, len(c.variant.columns))
	for i, typeName := range previousTypeNames {
		correctIndex := c.typeNamesIndex[typeName]

		sortedDiscriminatorMap[i] = uint8(correctIndex)
		sortedColumns[correctIndex] = c.variant.columns[i]
	}
	c.variant.columns = sortedColumns

	for i := range c.variant.discriminators {
		if c.variant.discriminators[i] == NullVariantDiscriminator {
			continue
		}

		c.variant.discriminators[i] = sortedDiscriminatorMap[c.variant.discriminators[i]]
	}
}

func (c *Dynamic) encodeHeader(buffer *proto.Buffer) {
	c.sortColumnsForEncoding()

	buffer.PutUInt64(SupportedDynamicSerializationVersion)
	buffer.PutUVarInt(uint64(c.maxTypes))
	buffer.PutUVarInt(uint64(c.totalTypes))

	for _, typeName := range c.typeNames {
		if typeName == "SharedVariant" {
			// SharedVariant is implicitly present in Dynamic, do not append to type names
			continue
		}

		buffer.PutString(typeName)
	}

	c.variant.encodeHeader(buffer)
}

func (c *Dynamic) encodeData(buffer *proto.Buffer) {
	c.variant.encodeData(buffer)
}

func (c *Dynamic) WriteStatePrefix(buffer *proto.Buffer) error {
	c.encodeHeader(buffer)

	return nil
}

func (c *Dynamic) Encode(buffer *proto.Buffer) {
	c.encodeData(buffer)
}

func (c *Dynamic) ScanType() reflect.Type {
	return scanTypeDynamic
}

func (c *Dynamic) Reset() {
	c.variant.Reset()
}

func (c *Dynamic) decodeHeader(reader *proto.Reader) error {
	dynamicSerializationVersion, err := reader.UInt64()
	if err != nil {
		return fmt.Errorf("failed to read dynamic serialization version: %w", err)
	}

	if dynamicSerializationVersion != SupportedDynamicSerializationVersion {
		return fmt.Errorf("unsupported dynamic serialization version: %d", dynamicSerializationVersion)
	}

	maxTypes, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read max types for dynamic column: %w", err)
	}
	c.maxTypes = uint8(maxTypes)

	totalTypes, err := reader.UVarInt()
	if err != nil {
		return fmt.Errorf("failed to read total types for dynamic column: %w", err)
	}

	sortedTypeNames := make([]string, 0, totalTypes+1)
	for i := uint64(0); i < totalTypes; i++ {
		typeName, err := reader.Str()
		if err != nil {
			return fmt.Errorf("failed to read type name at index %d for dynamic column: %w", i, err)
		}

		sortedTypeNames = append(sortedTypeNames, typeName)
	}

	sortedTypeNames = append(sortedTypeNames, "SharedVariant")
	slices.Sort(sortedTypeNames) // Re-sort after adding SharedVariant

	c.typeNames = make([]string, 0, len(sortedTypeNames))
	c.typeNamesIndex = make(map[string]int, len(sortedTypeNames))
	c.variant.columns = make([]Interface, 0, len(sortedTypeNames))
	c.variant.columnTypeIndex = make(map[string]uint8, len(sortedTypeNames))

	for _, typeName := range sortedTypeNames {
		col, err := Type(typeName).Column("", c.tz)
		if err != nil {
			return fmt.Errorf("failed to add dynamic column with type %s: %w", typeName, err)
		}

		c.addColumn(col)
	}

	c.totalTypes = uint8(totalTypes) // Reset to server's totalTypes

	err = c.variant.decodeHeader(reader)
	if err != nil {
		return fmt.Errorf("failed to decode variant header: %w", err)
	}

	return nil
}

func (c *Dynamic) decodeData(reader *proto.Reader, rows int) error {
	err := c.variant.decodeData(reader, rows)
	if err != nil {
		return fmt.Errorf("failed to decode variant data: %w", err)
	}

	return nil
}

func (c *Dynamic) ReadStatePrefix(reader *proto.Reader) error {
	err := c.decodeHeader(reader)
	if err != nil {
		return fmt.Errorf("failed to decode dynamic header: %w", err)
	}

	return nil
}

func (c *Dynamic) Decode(reader *proto.Reader, rows int) error {
	err := c.decodeData(reader, rows)
	if err != nil {
		return fmt.Errorf("failed to decode dynamic data: %w", err)
	}

	return nil
}
