package clickhouse_api

import (
	"context"
	"fmt"
	"reflect"
)

func DynamicScan() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	const query = `
	SELECT
		   1     AS Col1
		, 'Text' AS Col2
	`
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return err
	}
	var (
		columnTypes = rows.ColumnTypes()
		vars        = make([]any, len(columnTypes))
	)
	for i := range columnTypes {
		vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
	}
	for rows.Next() {
		if err := rows.Scan(vars...); err != nil {
			return err
		}
		for _, v := range vars {
			switch v := v.(type) {
			case *string:
				fmt.Println(*v)
			case *uint8:
				fmt.Println(*v)
			}
		}
	}
	return nil
}
