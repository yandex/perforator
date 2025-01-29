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

package main

import (
	"context"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	_ "github.com/mkevac/debugcharts"
)

type App struct {
	conn   driver.Conn
	signal chan os.Signal
}

func (app *App) invalidPrepare() {
	var i int
	for range time.Tick(time.Minute) {
		i++
		switch {
		case i%2 == 0:
			app.conn.PrepareBatch(context.Background(), "INSERT INTO x")
		default:
			batch, err := app.conn.PrepareBatch(context.Background(), "INSERT INTO stress")
			if err != nil {
				log.Fatal(err)
			}
			batch.Append(1, 1, 1, 1, 1)
		}
	}
}

func (app *App) worker() {
	for range time.Tick(time.Second) {
		app.batch()
	}
}

func (app *App) batch() {
	batch, err := app.conn.PrepareBatch(context.Background(), "INSERT INTO stress")
	if err != nil {
		log.Fatal("PrepareBatch", err)
	}
	for i := 0; i < 150_000; i++ {
		err := batch.Append(
			uint8(1),
			uuid.New(),
			time.Now(),
			[][]time.Time{
				[]time.Time{
					time.Now(),
					time.Now(),
				},
				[]time.Time{
					time.Now(),
					time.Now(),
				},
				[]time.Time{
					time.Now(),
					time.Now(),
				},
			},
			map[string]string{
				"key":  "value",
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
				"key4": "value4",
				"key5": "value5",
				"key6": "value6",
			},
		)
		if err != nil {
			log.Fatal("Append", err)
		}
	}
	if err := batch.Send(); err != nil {
		log.Fatal("Send", err)
	}
}

const ddl = `
CREATE TABLE stress (
	  Col1 UInt8
	, Col2 UUID
	, Col3 DateTime
	, Col4 Array(Array(DateTime))
	, Col5 Map(String, String)
) Engine Null
`

// http://127.0.0.1:8080/debug/pprof/
// http://127.0.0.1:8080/debug/charts/
func main() {
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	conn, err := clickhouse_tests.GetConnectionWithOptions(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		MaxOpenConns:    20,
		MaxIdleConns:    15,
		ConnMaxLifetime: 3 * time.Minute,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		// Debug: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := conn.Exec(context.Background(), "DROP TABLE IF EXISTS stress"); err != nil {
		log.Fatal(err)
	}
	if err := conn.Exec(context.Background(), ddl); err != nil {
		log.Fatal(err)
	}
	var (
		app = App{
			conn:   conn,
			signal: make(chan os.Signal),
		}
		signals = []os.Signal{
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGKILL,
		}
	)
	go app.invalidPrepare()
	for i := 0; i < 20; i++ {
		go app.worker()
	}
	signal.Notify(app.signal, signals...)
	{
		signal := <-app.signal
		{
			log.Println("got signal:", signal)
		}
		conn.Exec(context.Background(), "DROP TABLE IF EXISTS stress")
		os.Exit(0)
	}
}
