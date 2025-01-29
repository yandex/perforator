/*
   Copyright 2020 YANDEX LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package checkers

import (
	"context"
	"database/sql"
)

// Check executes specified query on specified database pool. Query must return single boolean
// value that signals if that pool is connected to primary or not. All errors are returned as is.
func Check(ctx context.Context, db *sql.DB, query string) (bool, error) {
	row := db.QueryRowContext(ctx, query)
	var primary bool
	if err := row.Scan(&primary); err != nil {
		return false, err
	}

	return primary, nil
}
