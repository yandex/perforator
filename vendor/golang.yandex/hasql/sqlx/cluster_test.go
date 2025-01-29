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

package hasql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCluster(t *testing.T) {
	checker := func(_ context.Context, _ *sql.DB) (bool, error) { return false, nil }

	t.Run("Works", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		require.NotNil(t, db)

		mock.ExpectClose()
		defer func() { assert.NoError(t, mock.ExpectationsWereMet()) }()

		node := NewNode("fake.addr", sqlx.NewDb(db, "sqlmock"))
		cl, err := NewCluster([]Node{node}, checker)
		require.NoError(t, err)
		require.NotNil(t, cl)
		defer func() { require.NoError(t, cl.Close()) }()

		require.Len(t, cl.Nodes(), 1)
		require.Equal(t, node, cl.Nodes()[0])
	})

	t.Run("Fails", func(t *testing.T) {
		cl, err := NewCluster(nil, checker)
		require.Error(t, err)
		require.Nil(t, cl)
	})
}
