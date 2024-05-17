package spectator

import (
	"fmt"
	"github.com/Netflix/spectator-go/spectator/logger"
	"github.com/Netflix/spectator-go/spectator/writer"
	"os"
)

// Config represents the Registry's configuration.
type Config struct {
	location   string
	commonTags map[string]string
	log        logger.Logger
}

// NewConfig creates a new configuration with the provided location, common tags, and logger. All fields are optional.
// Possible values for location are:
//   - `none`: Configures a no-op writer that does nothing. Can be used to disable metrics collection.
//   - `stdout`: Writes metrics to stdout.
//   - `stderr`: Writes metrics to stderr.
//   - `memory`: Writes metrics to memory. Useful for testing.
//   - `file:///path/to/file`: Writes metrics to a file.
//   - `unix:///path/to/socket`: Writes metrics to a Unix domain socket.
//   - `udp://host:port`: Writes metrics to a UDP socket.
//
// If location is not provided, the default value `udp://127.0.0.1:1234` will be used
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
		// Atlas doesn't support empty values for tags
		if v != "" {
			mergedTags[k] = v
		}
	}

	// merge common tags with env var tags. Env var tags take precedence
	for k, v := range tagsFromEnvVars() {
		// Atlas doesn't support empty values for tags
		if v != "" {
			mergedTags[k] = v
		}
	}
	return mergedTags
}

func calculateLocation(location string) (string, error) {
	if location != "" && !writer.IsValidOutputLocation(location) {
		return "", fmt.Errorf("invalid spectatord location: %s", location)
	}

	if override, ok := os.LookupEnv("SPECTATOR_OUTPUT_LOCATION"); ok {
		if !writer.IsValidOutputLocation(override) {
			return "", fmt.Errorf("SPECTATOR_OUTPUT_LOCATION is invalid: %s", override)
		}
		location = override
	}

	if location == "" { // use the default if there is no location or override
		location = "udp://127.0.0.1:1234"
	}

	return location, nil
}
