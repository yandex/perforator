package std

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/paulmach/orb"
)

func GeoInsertRead() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, clickhouse.Settings{
		"allow_experimental_geo_types": 1,
	}, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.ExecContext(ctx, "DROP TABLE IF EXISTS example")

	_, err = conn.ExecContext(ctx, `
		CREATE TABLE example (
				point Point,
				ring Ring,
				lineString LineString,
				polygon Polygon,
				mPolygon MultiPolygon,
				mLineString MultiLineString
			)
			Engine Memory
		`)
	if err != nil {
		return err
	}

	scope, err := conn.Begin()
	if err != nil {
		return err
	}
	batch, err := scope.Prepare("INSERT INTO example")
	if err != nil {
		return err
	}

	_, err = batch.Exec(
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
	)
	if err != nil {
		return err
	}

	if err = scope.Commit(); err != nil {
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

	if err = conn.QueryRow("SELECT * FROM example").Scan(&point, &ring, &lineString, &polygon, &mPolygon, &mLineString); err != nil {
		return err
	}
	fmt.Printf("point=%v, ring=%v, lineString=%v, polygon=%v, mPolygon=%v, mLineString=%v\n", point, ring, lineString, polygon, mPolygon, mLineString)
	return nil
}
