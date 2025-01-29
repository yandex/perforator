package testutils

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/xerrors"

	"github.com/yandex/perforator/perforator/pkg/postgres"
)

func DefaultTestConfig() (postgres.Config, error) {
	cfg := postgres.Config{}

	host, err := getEnvVar("POSTGRES_RECIPE_HOST")
	if err != nil {
		return postgres.Config{}, err
	}
	portString, err := getEnvVar("POSTGRES_RECIPE_PORT")
	if err != nil {
		return postgres.Config{}, err
	}
	port, err := stringToUint16(portString)
	if err != nil {
		return postgres.Config{}, fmt.Errorf("couldn't parse port: %w", err)
	}

	cfg.Endpoints = []postgres.Endpoint{
		{
			Host: host,
			Port: port,
		},
	}

	cfg.AuthConfig.User, err = getEnvVar("POSTGRES_RECIPE_USER")
	if err != nil {
		return postgres.Config{}, err
	}

	cfg.DB, err = getEnvVar("POSTGRES_RECIPE_DBNAME")
	if err != nil {
		return postgres.Config{}, err
	}

	cfg.AuthConfig.PasswordEnv = "POSTGRES_PASSWORD"
	cfg.SSLMode = postgres.SSLModeDisable

	return cfg, nil
}

func getEnvVar(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", xerrors.Errorf("%s is missing", key)
	}
	return val, nil
}

func stringToUint16(s string) (uint16, error) {
	var base = 10
	var size = 16

	ui64, err := strconv.ParseUint(s, base, size)
	if err != nil {
		return 0, err
	}
	ui16 := uint16(ui64)

	return ui16, nil
}
