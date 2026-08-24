// Package catalog contains catalog location and access helpers.
package catalog

import (
	"errors"
	"os"
)

const HomeEnvironment = "PAPERSPLZ_HOME"

var ErrHomeRequired = errors.New("catalog home is required; use --home PATH or set PAPERSPLZ_HOME")

// ResolveHome selects a catalog path. An explicit non-empty path takes
// precedence over PAPERSPLZ_HOME.
func ResolveHome(explicit string, lookupEnv func(string) (string, bool)) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if home, ok := lookupEnv(HomeEnvironment); ok && home != "" {
		return home, nil
	}
	return "", ErrHomeRequired
}

// ResolveHomeFromEnvironment resolves a catalog using the process environment.
func ResolveHomeFromEnvironment(explicit string) (string, error) {
	return ResolveHome(explicit, os.LookupEnv)
}
