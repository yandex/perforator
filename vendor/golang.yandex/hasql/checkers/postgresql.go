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

// PostgreSQL checks whether PostgreSQL server is primary or not.
func PostgreSQL(ctx context.Context, db *sql.DB) (bool, error) {
	return Check(ctx, db, "SELECT NOT pg_is_in_recovery()")
}
