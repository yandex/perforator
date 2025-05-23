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

package clickhouse

import "github.com/ClickHouse/clickhouse-go/v2/lib/chcol"

// Re-export chcol types/funcs to top level clickhouse package

type (
	// Variant represents a ClickHouse Variant type that can hold multiple possible types
	Variant = chcol.Variant
	// Dynamic is an alias for the Variant type
	Dynamic = chcol.Dynamic
	// JSON represents a ClickHouse JSON type that can hold multiple possible types
	JSON = chcol.JSON

	// JSONSerializer interface allows a struct to be manually converted to an optimized JSON structure instead of relying
	// on recursive reflection.
	// Note that the struct must be a pointer in order for the interface to be matched, reflection will be used otherwise.
	JSONSerializer = chcol.JSONSerializer
	// JSONDeserializer interface allows a struct to load its data from an optimized JSON structure instead of relying
	// on recursive reflection to set its fields.
	JSONDeserializer = chcol.JSONDeserializer
)

// NewVariant creates a new Variant with the given value
func NewVariant(v any) Variant {
	return chcol.NewVariant(v)
}

// NewVariantWithType creates a new Variant with the given value and ClickHouse type
func NewVariantWithType(v any, chType string) Variant {
	return chcol.NewVariantWithType(v, chType)
}

// NewDynamic creates a new Dynamic with the given value
func NewDynamic(v any) Dynamic {
	return chcol.NewDynamic(v)
}

// NewDynamicWithType creates a new Dynamic with the given value and ClickHouse type
func NewDynamicWithType(v any, chType string) Dynamic {
	return chcol.NewDynamicWithType(v, chType)
}

// NewJSON creates a new empty JSON value
func NewJSON() *JSON {
	return chcol.NewJSON()
}

// ExtractJSONPathAs is a convenience function for asserting a path to a specific type.
// The underlying value is also extracted from its Dynamic wrapper if present.
func ExtractJSONPathAs[T any](o *JSON, path string) (valueAs T, ok bool) {
	return chcol.ExtractJSONPathAs[T](o, path)
}
