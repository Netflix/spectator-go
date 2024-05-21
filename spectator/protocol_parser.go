package spectator

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/meter"
	"strings"
)

// ParseProtocolLine parses a line of the spectator protocol. Utility exposed for testing.
func ParseProtocolLine(line string) (string, *meter.Id, string, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 3 {
		return "", nil, "", fmt.Errorf("invalid line format")
	}

	meterSymbol := parts[0]
	meterId := parts[1]
	value := parts[2]

	meterIdParts := strings.Split(meterId, ",")
	name := meterIdParts[0]

	tags := make(map[string]string)
	for _, tag := range meterIdParts[1:] {
		kv := strings.Split(tag, "=")
		if len(kv) != 2 {
			return "", nil, "", fmt.Errorf("invalid tag format")
		}
		tags[kv[0]] = kv[1]
	}

	return meterSymbol, meter.NewId(name, tags), value, nil
}
