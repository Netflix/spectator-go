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
//   - nf.container: NETFLIX_CONTAINER_NAME (injected by the platform with the container name),
//     falling back to TITUS_CONTAINER_NAME.
//   - nf.process: NETFLIX_PROCESS_NAME, set by the user for each process.
//
// The extracted variables are specific to a container or process and thus cannot be managed by a
// shared SpectatorD instance. Unset, empty and whitespace-only values are ignored.
func tagsFromEnvVars() map[string]string {
	tags := make(map[string]string)
	addNonEmpty(tags, "nf.container", "NETFLIX_CONTAINER_NAME", "TITUS_CONTAINER_NAME")
	addNonEmpty(tags, "nf.process", "NETFLIX_PROCESS_NAME")
	return tags
}
