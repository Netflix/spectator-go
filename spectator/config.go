package spectator

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"os"
)

// Config represents the Registry's configuration.
type Config struct {
	location   string
	commonTags map[string]string
	log        logger.Logger
}

// NewConfig creates a new configuration with the provided location, extra common tags, and logger. All fields are
// optional. The extra common tags are added to every metric, outside the common tags provided by spectatord.
//
// Possible values for location are:
//
//   - `none`   - Configure a no-op writer that does nothing. Can be used to disable metrics collection.
//   - `memory` - Write metrics to memory. Useful for testing.
//   - `stderr` - Write metrics to standard error.
//   - `stdout` - Write metrics to standard output.
//   - `unix`   - Write metrics to the default Unix Domain Socket. Useful for high-volume scenarios.
//   - `file:///path/to/file`   - Write metrics to a file.
//   - `udp://host:port`        - Write metrics to a UDP socket.
//   - `unix:///path/to/socket` - Write metrics to a Unix Domain Socket.
//
// If a location is not provided, or it is an empty string, then the default location of `udp://127.0.0.1:1234` will
// be used.
//
// The output location can be overridden by configuring an environment variable SPECTATOR_OUTPUT_LOCATION
// with one of the values listed above. Overriding the output location may be useful for integration testing.
func NewConfig(
	location string, // defaults to `udp://127.0.0.1:1234`
	commonTags map[string]string, // defaults to empty map
	log logger.Logger, // defaults to default logger
) (*Config, error) {
	location, err := calculateLocation(location)
	if err != nil {
		return nil, err
	}

	mergedTags := calculateCommonTags(commonTags)

	lg := calculateLogger(log)

	return &Config{
		location:   location,
		commonTags: mergedTags,
		log:        lg,
	}, nil
}

func calculateLogger(log logger.Logger) logger.Logger {
	lg := log
	if log == nil {
		lg = logger.NewDefaultLogger()
	}
	return lg
}

func calculateCommonTags(commonTags map[string]string) map[string]string {
	mergedTags := make(map[string]string)

	for k, v := range commonTags {
		// tag keys and values may not be empty strings
		if k != "" && v != "" {
			mergedTags[k] = v
		}
	}

	// merge common tags with env var tags; env var tags take precedence
	for k, v := range tagsFromEnvVars() {
		// env tags are validated to be non-empty
		mergedTags[k] = v
	}

	return mergedTags
}

func calculateLocation(location string) (string, error) {
	if location != "" && !writer.IsValidOutputLocation(location) {
		return "", fmt.Errorf("invalid spectatord output location: %s", location)
	}

	if override, ok := os.LookupEnv("SPECTATOR_OUTPUT_LOCATION"); ok {
		if !writer.IsValidOutputLocation(override) {
			return "", fmt.Errorf("SPECTATOR_OUTPUT_LOCATION is invalid: %s", override)
		}
		location = override
	}

	if location == "" { // use the default, if there is no location or override
		location = "udp://127.0.0.1:1234"
	}

	return location, nil
}
