package spectator

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"os"
)

// Config represents the Registry's configuration.
type Config struct {
	Location   string            `json:"sidecar.output-location"`
	CommonTags map[string]string `json:"sidecar.common-tags"`
	Log        logger.Logger
}

// GetLocation Checks the location provided in the config and the environment variable SPECTATOR_OUTPUT_LOCATION.
// If neither are valid, returns the default UDP address.
func (c *Config) GetLocation() string {
	configValue := c.Location
	envValue := os.Getenv("SPECTATOR_OUTPUT_LOCATION")

	if writer.ValidOutputLocation(configValue) {
		return configValue
	} else if writer.ValidOutputLocation(envValue) {
		return envValue
	} else {
		return "udp://127.0.0.1:1234"
	}
}
