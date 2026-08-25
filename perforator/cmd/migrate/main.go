package main

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

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

type tlsConfig struct {
	deprecatedInsecure  bool
	plaintext           bool
	skipTLSVerification bool
	serverCA            string
}

func (t *tlsConfig) validate(deprHandler func(string)) error {
	if t.deprecatedInsecure {
		deprHandler("--insecure is deprecated, use --plaintext or --tls-trust-all instead")
	}
	if t.plaintext && t.skipTLSVerification {
		return errors.New("--plaintext and --tls-trust-all are mutually exclusive")
	}
	if t.plaintext && t.serverCA != "" {
		return errors.New("--plaintext and --tls-ca are mutually exclusive")
	}
	if t.serverCA != "" && t.skipTLSVerification {
		return errors.New("--tls-ca and --tls-trust-all are mutually exclusive")
	}
	return nil
}

var (
	hosts        []string
	port         uint16
	database     string
	username     string
	password     string
	passwordFile string
	tls          tlsConfig

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

	resolvedPassword, err := resolvePassword(password, passwordFile)
	if err != nil {
		return err
	}

	return runMigrations(dbByCmd(cmd), hosts, port, database, username, resolvedPassword, tls, func(m *migrate.Migrate) error {
		return m.Migrate(uint(version))
	})
}

func forceCmdRunE(cmd *cobra.Command, args []string) error {
	version, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse version %s: %w", args[0], err)
	}

	resolvedPassword, err := resolvePassword(password, passwordFile)
	if err != nil {
		return err
	}

	return runMigrations(dbByCmd(cmd), hosts, port, database, username, resolvedPassword, tls, func(m *migrate.Migrate) error {
		return m.Force(int(version))
	})
}

func upCmdRunE(cmd *cobra.Command, _ []string) error {
	resolvedPassword, err := resolvePassword(password, passwordFile)
	if err != nil {
		return err
	}

	return runMigrations(dbByCmd(cmd), hosts, port, database, username, resolvedPassword, tls, func(m *migrate.Migrate) error {
		return m.Up()
	})
}

func resolvePassword(value, path string) (string, error) {
	if value != "" && path != "" {
		return "", errors.New("--pass and --pass-file are mutually exclusive")
	}
	if path == "" {
		return value, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read password file: %w", err)
	}

	result := strings.TrimSuffix(string(data), "\n")
	result = strings.TrimSuffix(result, "\r")
	return result, nil
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
			subcommand.Flags().StringVar(&passwordFile, "pass-file", "", "Path to a file containing the password")
			subcommand.Flags().BoolVar(&tls.deprecatedInsecure, "insecure", false, "(Deprecated) disable transport security")
			subcommand.Flags().BoolVar(&tls.plaintext, "plaintext", false, "Use plaintext connection")
			subcommand.Flags().BoolVar(&tls.skipTLSVerification, "tls-trust-all", false, "Skip TLS verification")
			subcommand.Flags().StringVar(&tls.serverCA, "tls-ca", "", "Path to CA certificate")
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
	tls tlsConfig,
	callback func(*migrate.Migrate) error,
) error {
	tlsErr := tls.validate(func(s string) { log.Printf("Warning: %s", s) })
	if tlsErr != nil {
		return fmt.Errorf("invalid tls configuration: %w", tlsErr)
	}
	errs := make([]error, 0)

	log.Printf("Starting migrations")

	for _, host := range hosts {
		mig, err := newMigrate(db, host, port, database, username, password, tls)

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
	tls tlsConfig,
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

	var queryParams []string

	switch db {
	case Clickhouse:
		queryParams = append(queryParams, "x-multi-statement=true")
		if tls.deprecatedInsecure || tls.plaintext {
			queryParams = append(queryParams, "secure=false")
		} else {
			queryParams = append(queryParams, "secure=true")
		}
		if tls.skipTLSVerification {
			queryParams = append(queryParams, "skip_verify=true")
		}
		if tls.serverCA != "" {
			return nil, errors.New("tls-ca is not supported for clickhouse")
		}
	case Postgres:
		sslmode := "require"

		if tls.deprecatedInsecure || tls.plaintext {
			sslmode = "disable"
		} else if tls.skipTLSVerification {
			// TODO: this case looks broken in postgres
			sslmode = "require"
		} else {
			if tls.serverCA == "" {
				queryParams = append(queryParams, "sslrootcert=system")
			} else {
				queryParams = append(queryParams, "sslrootcert=", tls.serverCA)
			}
		}

		queryParams = append(queryParams, fmt.Sprint("sslmode=", sslmode))
	}
	if len(queryParams) > 0 {
		uri += "?"
	}
	for i, qp := range queryParams {
		if i > 0 {
			uri += "&"
		}
		uri += qp
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, uri)
	if err != nil {
		return nil, err
	}
	m.Log = &logger{}

	return m, nil
}
