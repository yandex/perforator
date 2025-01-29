package main

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"net/url"

	"cuelang.org/go/pkg/strconv"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
)

type DB string

const (
	Clickhouse DB = "clickhouse"
	Postgres   DB = "postgresql"
)

var (
	defaultPorts = map[DB]uint16{
		Clickhouse: 9440,
		Postgres:   6432,
	}
)

//go:embed migrations/clickhouse/*.sql
var migrationsClickhouse embed.FS

//go:embed migrations/postgres/*.sql
var migrationsPostgres embed.FS

var (
	hosts    []string
	port     uint16
	database string
	username string
	password string
	insecure bool

	rootCmd = &cobra.Command{
		Use:           "migrate",
		Short:         "Run migrations on hosts",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	clickhouseCmd = &cobra.Command{
		Use:   "clickhouse {up | force | migrate}",
		Short: "Run migrations on clickhouse hosts",
	}

	postgresCmd = &cobra.Command{
		Use:   "postgres {up | force | migrate}",
		Short: "Run migrations on postgres hosts",
	}
)

func migrateCmdRunE(cmd *cobra.Command, args []string) error {
	version, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version %s: %w", args[0], err)
	}

	return runMigrations(dbByCmd(cmd), hosts, port, database, username, password, insecure, func(m *migrate.Migrate) error {
		return m.Migrate(uint(version))
	})
}

func forceCmdRunE(cmd *cobra.Command, args []string) error {
	version, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version %s: %w", args[0], err)
	}

	return runMigrations(dbByCmd(cmd), hosts, port, database, username, password, insecure, func(m *migrate.Migrate) error {
		return m.Force(int(version))
	})
}

func upCmdRunE(cmd *cobra.Command, _ []string) error {
	return runMigrations(dbByCmd(cmd), hosts, port, database, username, password, insecure, func(m *migrate.Migrate) error {
		return m.Up()
	})
}

func dbByCmd(cmd *cobra.Command) DB {
	switch cmd.Parent() {
	case postgresCmd:
		return Postgres
	case clickhouseCmd:
		return Clickhouse
	default:
		panic("cannot get db from cmd")
	}
}

func init() {
	rootCmd.AddCommand(postgresCmd)
	rootCmd.AddCommand(clickhouseCmd)
	cobrabuildinfo.Init(rootCmd)

	for _, command := range []*cobra.Command{postgresCmd, clickhouseCmd} {
		command.AddCommand(&cobra.Command{
			Use:   "force <version>",
			Short: "Force migration version",
			RunE:  forceCmdRunE,
		})
		command.AddCommand(&cobra.Command{
			Use:   "up",
			Short: "Upgrade to the newest version",
			RunE:  upCmdRunE,
		})
		command.AddCommand(&cobra.Command{
			Use:   "migrate <version>",
			Short: "Migrate to certain version",
			RunE:  migrateCmdRunE,
		})

		for _, subcommand := range command.Commands() {
			subcommand.Flags().StringSliceVar(&hosts, "hosts", []string{}, "Hosts, separated by comma. For postgres specify only master.")
			subcommand.Flags().Uint16VarP(&port, "port", "p", 0, "Port")
			subcommand.Flags().StringVar(&database, "db", "perforator", "Database name")
			subcommand.Flags().StringVar(&username, "user", "perforator", "Username")
			subcommand.Flags().StringVar(&password, "pass", "", "Password")
			subcommand.Flags().BoolVar(&insecure, "insecure", false, "Disable TLS")
		}
	}
}

func main() {
	maxprocs.Adjust()

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runMigrations(
	db DB,
	hosts []string,
	port uint16,
	database string,
	username string,
	password string,
	insecure bool,
	callback func(*migrate.Migrate) error,
) error {
	errs := make([]error, 0)

	log.Printf("Starting migrations")

	for _, host := range hosts {
		mig, err := newMigrate(db, host, port, database, username, password, insecure)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to migrate host %s: %w", host, err))
			continue
		}

		err = callback(mig)
		if err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Printf("Host %s is already up to date\n", host)
			} else {
				errs = append(errs, fmt.Errorf("failed to migrate host %s: %w", host, err))
			}
			continue
		}

		log.Printf("Successfully migrated host %s\n", host)
	}

	return errors.Join(errs...)
}

func newMigrate(
	db DB,
	host string,
	port uint16,
	database string,
	username string,
	password string,
	insecure bool,
) (*migrate.Migrate, error) {
	if port == 0 {
		port = defaultPorts[db]
	}

	var path string
	var migrations embed.FS

	switch db {
	case Clickhouse:
		path = "migrations/clickhouse"
		migrations = migrationsClickhouse
	case Postgres:
		path = "migrations/postgres"
		migrations = migrationsPostgres
	}

	d, err := iofs.New(migrations, path)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("%s://%s@%s:%d/%s",
		string(db),
		url.UserPassword(username, password).String(),
		host,
		port,
		database,
	)

	switch db {
	case Clickhouse:
		uri += fmt.Sprintf("?x-multi-statement=true&secure=%s", strconv.FormatBool(!insecure))
	case Postgres:
		sslmode := "require"

		if insecure {
			sslmode = "disable"
		}

		uri += "?sslmode=" + sslmode
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, uri)
	if err != nil {
		return nil, err
	}
	m.Log = &logger{}

	return m, nil
}
