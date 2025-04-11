package spectator

import (
	"os"
	"strings"
)

func addNonEmpty(tags map[string]string, tag string, envVars ...string) {
	for _, envVar := range envVars {
		value, exists := os.LookupEnv(envVar)
		if !exists {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) != 0 {
			tags[tag] = value
			break
		}
	}
}

// tagsFromEnvVars Extract common infrastructure tags from the Netflix environment variables.
//
// The extracted variables are specific to a process and thus cannot be managed by a shared
// SpectatorD instance.
func tagsFromEnvVars() map[string]string {
	tags := make(map[string]string)
	addNonEmpty(tags, "nf.container", "TITUS_CONTAINER_NAME")
	addNonEmpty(tags, "nf.process", "NETFLIX_PROCESS_NAME")
	return tags
}
