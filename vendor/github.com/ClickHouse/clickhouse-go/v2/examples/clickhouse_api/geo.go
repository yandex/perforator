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
				lineString LineString,
				polygon Polygon,
				mPolygon MultiPolygon,
				mLineString MultiLineString
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
		orb.LineString{
			orb.Point{1, 2},
			orb.Point{3, 4},
			orb.Point{5, 6},
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
		orb.MultiLineString{
			orb.LineString{
				orb.Point{1, 2},
				orb.Point{3, 4},
			},
			orb.LineString{
				orb.Point{5, 6},
				orb.Point{7, 8},
			},
		},
	); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var (
		point       orb.Point
		ring        orb.Ring
		lineString  orb.LineString
		polygon     orb.Polygon
		mPolygon    orb.MultiPolygon
		mLineString orb.MultiLineString
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&point, &ring, &lineString, &polygon, &mPolygon, &mLineString); err != nil {
		return err
	}
	fmt.Printf("point=%v, ring=%v, lineString=%v, polygon=%v, mPolygon=%v, mLineString=%v\n", point, ring, lineString, polygon, mPolygon, mLineString)
	return nil
}
