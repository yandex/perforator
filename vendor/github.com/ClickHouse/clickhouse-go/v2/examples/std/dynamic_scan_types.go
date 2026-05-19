package std

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"reflect"
)

func DynamicScan() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	const query = `
	SELECT
		   1     AS Col1
		, 'Text' AS Col2
	`
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		return err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	vars := make([]any, len(columnTypes))
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
