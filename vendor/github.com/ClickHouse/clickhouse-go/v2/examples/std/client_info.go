package std

import "database/sql"

func ClientInfo() error {
	db, err := sql.Open("clickhouse", "clickhouse://default@127.0.0.1/default?client_info_product[my-app]=0.1")
	if err != nil {
		return err
	}

	_, err = db.Exec("SELECT 1")
	return err
}
