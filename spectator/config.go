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

// NewConfigWithBuffer creates a new configuration with the provided location, extra common tags, logger, bufferSize,
// and flushInterval. This factory function should be used when you need additional performance when publishing metrics.
//
// There are two buffer implementations that can be selected, when a bufferSize > 0 is configured:
//
//   - LineBuffer (bufferSize <= 65536), which is a single string buffer, protected by a mutex, that offers
//     write performance up to ~1M lines/sec (spectatord maximum), with a latency per write ranging from
//     0.1 to 32 us, depending upon the number of threads in use.
//
//     Metrics are flushed from the buffer when an overflow occurs, and periodically by a timer, according to the
//     flush interval. Thus, if there are periods of time when metric publishing is slow, metrics will still be
//     delivered from the buffer on time. Note that the spectatord publish interval is every 5 seconds, which is
//     a good choice for this configuration. This buffer will block, and it will not drop lines.
//
//     The LineBuffer reports metrics, which can be used to monitor buffer performance:
//
//       - spectator-go.lineBuffer.bytesWritten - A counter reporting bytes/sec written to spectatord.
//       - spectator-go.lineBuffer.overflows    - A counter reporting overflows/sec, which are flushes before the interval.
//
//     Example configuration:
//
//       config, _ := NewConfigWithBuffer("udp", nil, nil, 61440, 5*time.Second)
//
//   - LowLatencyBuffer (bufferSize > 65536), which builds arrays of buffers that are optimized for introducing
//     the least amount of latency in highly multithreaded applications that record many metrics. It offers write
//     performance up to ~1 M lines/sec (spectatord maximum), with a latency per write ranging from 0.6 to 7 us,
//     depending upon the number of threads in use.
//
//     This is achieved by spreading data access across a number of different mutexes, and only writing buffers from
//     a goroutine that runs periodically, according to the flush interval. There is a front buffer and a back buffer,
//     and these are rotated during the periodic flush. The inactive buffer is flushed, while the active buffer
//     continues to receive metric writes from the application. Within each buffer, there are numCPU shards, and each
//     buffer shard has N chunks, where a chunk is set to 60KB, to allow the data to fit within the spectatord socket
//     buffers with room for one last protocol line. This buffer will not block, and it can drop lines, if it overflows.
//
//     As a sizing example, if you have an 8 CPU system, and you want to allocate 5 MB to each buffer shard, and
//     there are two buffers (front and back), then you need to configure a buffer size of 83,886,080 bytes. Each
//     buffer shard will have 85 chunks, each of which is protected by a separate mutex.
//
//       2 buffers (front/back) * 8 CPU (shard count) * 5,242,880 bytes/shard *  = 83,886,080 bytes total
//
//     Pairing this with a 1-second flush interval will result in a configuration that can handle ~85K lines/sec writes
//     to spectatord. Note that the spectatord publish interval is every 5 seconds, so you have some room to experiment
//     with different buffer sizes and publish intervals.
//
//     While the bufferSize can be set as low as 65537, it will guarantee a minimum size of 2 * CPU * 60KB, to ensure
//     that there is always at least 1 chunk per shard. On a system with 1 CPU, this will be 122,880 bytes, and on a
//     system with 4 CPU, this will be 491,520 bytes.
//
//     The LowLatencyBuffer reports metrics, which can be used to monitor buffer performance:
//
//       - spectator-go.lowLatencyBuffer.bytesWritten - A counter reporting bytes/sec written to spectatord.
//       - spectator-go.lowLatencyBuffer.overflows    - A counter reporting overflows/sec, which are drops.
//       - spectator-go.lowLatencyBuffer.pctUsage     - A gauge reporting the percent usage of the buffers.
//
//     When using the LowLatencyBuffer, it is recommended to watch the spectatord.parsedCount metric, to ensure that
//     you have sufficient headroom against the maximum data ingestion rate of ~1M lines/sec for spectatord.
//
//     Example configuration:
//
//       config, _ := NewConfigWithBuffer("udp", nil, nil, 83886080, 1*time.Second)
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
