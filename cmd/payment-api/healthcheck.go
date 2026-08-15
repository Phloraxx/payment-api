package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultHealthcheckURL = "http://127.0.0.1:3000/api/health"

// runStandaloneHealthcheck handles the Docker healthcheck before PocketBase is
// constructed. Bootstrapping a second PocketBase process against the live
// SQLite files can interfere with WAL/SHM ownership of the serving process.
func runStandaloneHealthcheck(args []string) (bool, error) {
	if len(args) == 0 || args[0] != "healthcheck" {
		return false, nil
	}

	flags := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	endpoint := flags.String("url", defaultHealthcheckURL, "health endpoint URL")
	if err := flags.Parse(args[1:]); err != nil {
		return true, err
	}
	if flags.NArg() != 0 {
		return true, fmt.Errorf("healthcheck accepts no positional arguments")
	}

	return true, checkHealthEndpoint(*endpoint)
}

func checkHealthEndpoint(endpoint string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("health endpoint returned HTTP %d", response.StatusCode)
	}
	return nil
}
