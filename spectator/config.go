package spectator

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"os"
	"time"
)

// Config represents the Registry's configuration.
type Config struct {
	location        string
	extraCommonTags map[string]string
	log             logger.Logger
	bufferSize      int
	flushInterval   time.Duration
}

// NewConfig creates a new configuration with the provided location, extra common tags, and logger. All fields are
// optional. The extra common tags are added to every metric, on top of the common tags provided by spectatord.
//
// Possible values for location are:
//
//   - `""`     - Empty string will default to `udp`.
//   - `none`   - Configure a no-op writer that does nothing. Can be used to disable metrics collection.
//   - `memory` - Write metrics to memory. Useful for testing.
//   - `stderr` - Write metrics to standard error.
//   - `stdout` - Write metrics to standard output.
//   - `udp`    - Write metrics to the default spectatord UDP port. This is the default value.
//   - `unix`   - Write metrics to the default spectatord Unix Domain Socket. Useful for high-volume scenarios.
//   - `file:///path/to/file`   - Write metrics to a file.
//   - `udp://host:port`        - Write metrics to a UDP socket.
//   - `unix:///path/to/socket` - Write metrics to a Unix Domain Socket.
//
// The output location can be overridden by configuring an environment variable SPECTATOR_OUTPUT_LOCATION
// with one of the values listed above. Overriding the output location may be useful for integration testing.
func NewConfig(
	location string, // defaults to `udp`
	extraCommonTags map[string]string, // defaults to empty map
	log logger.Logger, // defaults to default logger
) (*Config, error) {
	return NewConfigWithBuffer(location, extraCommonTags, log, 0, 5 * time.Second)
}

// NewConfigWithBuffer creates a new configuration with the provided location, extra common tags, logger,
// bufferSize, and flushInterval. This factory function should be used when you need additional performance
// when publishing metrics.
//
// Three modes of operation are available, for applications that operate at different scales:
//
//   - Small. No buffer (size 0 bytes). Write immediately to the socket upon every metric update, up to ~150K
//     lines/sec, with delays from 2 to 450 us, depending on thread count. No metrics are dropped, due to mutex
//     locks.
//   - Medium. LineBuffer (size <= 65536 bytes), which writes to the socket upon overflow, or upon a flush
//     interval, up to ~1M lines/sec, with delays from 0.1 to 32 us, depending on thread count. No metrics are
//     dropped. Status metrics are published to monitor usage.
//   - Large. LowLatencyBuffer (size > 65536 bytes), which writes to the socket on a flush interval, up to ~1M
//     lines/sec, with delays from 0.6 to 7 us, depending on thread count. The true minimum size is 2 * CPU *
//     60KB, or 122,880 bytes for 1 CPU. Metrics may be dropped. Status metrics are published to monitor usage.
//
// The buffers are available for the UdpWriter and the UnixWriter.
//
// See https://netflix.github.io/atlas-docs/spectator/lang/go/usage/#buffers for a more detailed explanation.
//
func NewConfigWithBuffer(
	location string, // defaults to `udp`
	extraCommonTags map[string]string, // defaults to empty map
	log logger.Logger, // defaults to default logger
	bufferSize int, // defaults to 0 (disabled)
	flushInterval time.Duration, // defaults to 5 seconds
) (*Config, error) {
	location, err := calculateLocation(location)
	if err != nil {
		return nil, err
	}

	mergedTags := calculateExtraCommonTags(extraCommonTags)

	return &Config{
		location:        location,
		extraCommonTags: mergedTags,
		log:             calculateLogger(log),
		bufferSize:      bufferSize,
		flushInterval:   flushInterval,
	}, nil
}

func calculateLogger(log logger.Logger) logger.Logger {
	if log == nil {
		return logger.NewDefaultLogger()
	} else {
		return log
	}
}

func calculateExtraCommonTags(extraCommonTags map[string]string) map[string]string {
	mergedTags := make(map[string]string)

	for k, v := range extraCommonTags {
		// tag keys and values may not be empty strings
		if k != "" && v != "" {
			mergedTags[k] = v
		}
	}

	// merge extra common tags with select env var tags; env var tags take precedence
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
		location = "udp"
	}

	return location, nil
}
