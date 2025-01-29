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

package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/paulmach/orb"
)

func GeoInsertRead() error {
	conn, err := GetNativeConnection(clickhouse.Settings{
		"allow_experimental_geo_types": 1,
	}, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
				point Point,
				ring Ring,
				polygon Polygon,
				mPolygon MultiPolygon
			) 
			Engine Memory
		`); err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	if err = batch.Append(
		orb.Point{11, 22},
		orb.Ring{
			orb.Point{1, 2},
			orb.Point{1, 2},
		},
		orb.Polygon{
			orb.Ring{
				orb.Point{1, 2},
				orb.Point{12, 2},
			},
			orb.Ring{
				orb.Point{11, 2},
				orb.Point{1, 12},
			},
		},
		orb.MultiPolygon{
			orb.Polygon{
				orb.Ring{
					orb.Point{1, 2},
					orb.Point{12, 2},
				},
				orb.Ring{
					orb.Point{11, 2},
					orb.Point{1, 12},
				},
			},
			orb.Polygon{
				orb.Ring{
					orb.Point{1, 2},
					orb.Point{12, 2},
				},
				orb.Ring{
					orb.Point{11, 2},
					orb.Point{1, 12},
				},
			},
		},
	); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var (
		point    orb.Point
		ring     orb.Ring
		polygon  orb.Polygon
		mPolygon orb.MultiPolygon
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&point, &ring, &polygon, &mPolygon); err != nil {
		return err
	}
	fmt.Printf("point=%v, ring=%v, polygon=%v, mPoilygon=%v\n", point, ring, polygon, mPolygon)
	return nil
}
