package client

import (
	"fmt"
	"os"
	"strconv"
)

const perforatorEndpointEnv = `PERFORATOR_ENDPOINT`
const perforatorSecurityLevelEnv = `PERFORATOR_SECURE`

func getDefaultPerforatorEndpoint() (endpoint, error) {
	var e endpoint

	if url, ok := os.LookupEnv(perforatorEndpointEnv); !ok {
		return e, fmt.Errorf("environment variable %s is not set", perforatorEndpointEnv)
	} else {
		e.url = url
	}

	if level, ok := os.LookupEnv(perforatorSecurityLevelEnv); ok {
		secure, err := strconv.ParseBool(level)
		if err != nil {
			return e, fmt.Errorf(
				"failed to parse %s environment variable: expected bool, found %s",
				perforatorSecurityLevelEnv, level,
			)
		}
		e.secure = secure
	}

	return e, nil
}
