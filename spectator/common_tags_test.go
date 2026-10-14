package spectator

import (
	"os"
	"testing"
)

func TestAddNonEmptyWithExistingEnvVar(t *testing.T) {
	_ = os.Setenv("EXISTING_ENV_VAR", "test_value")
	defer os.Unsetenv("EXISTING_ENV_VAR")

	tags := make(map[string]string)
	addNonEmpty(tags, "test_tag", "EXISTING_ENV_VAR")

	if tags["test_tag"] != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", tags["test_tag"])
	}
}

func TestAddNonEmptyWithNonExistingEnvVar(t *testing.T) {
	tags := make(map[string]string)
	addNonEmpty(tags, "test_tag", "NON_EXISTING_ENV_VAR")

	if _, ok := tags["test_tag"]; ok {
		t.Errorf("Expected tag not to be set")
	}
}

func TestAddNonEmptyWithEmptyEnvVar(t *testing.T) {
	_ = os.Setenv("EMPTY_ENV_VAR", "")
	defer os.Unsetenv("EMPTY_ENV_VAR")

	tags := make(map[string]string)
	addNonEmpty(tags, "test_tag", "EMPTY_ENV_VAR")

	if _, ok := tags["test_tag"]; ok {
		t.Errorf("Expected tag not to be set")
	}
}

func TestTagsFromEnvVars(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "container_name")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "process_name")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")
	// An injected value would otherwise beat TITUS_CONTAINER_NAME. t.Setenv restores it after the test, and
	// Unsetenv makes it absent rather than empty.
	t.Setenv("NETFLIX_CONTAINER_NAME", "")
	_ = os.Unsetenv("NETFLIX_CONTAINER_NAME")

	tags := tagsFromEnvVars()

	if tags["nf.container"] != "container_name" {
		t.Errorf("Expected 'container_name', got '%s'", tags["nf.container"])
	}

	if tags["nf.process"] != "process_name" {
		t.Errorf("Expected 'process_name', got '%s'", tags["nf.process"])
	}
}

func TestAddNonEmptyFirstNonEmptyWins(t *testing.T) {
	t.Setenv("FIRST_ENV_VAR", " \t ")
	t.Setenv("SECOND_ENV_VAR", " second ")
	t.Setenv("THIRD_ENV_VAR", "third")

	tags := make(map[string]string)
	addNonEmpty(tags, "test_tag", "FIRST_ENV_VAR", "SECOND_ENV_VAR", "THIRD_ENV_VAR")

	if tags["test_tag"] != "second" {
		t.Errorf("Expected 'second', got '%s'", tags["test_tag"])
	}
}

func TestTagsFromEnvVarsPrefersNetflixContainerName(t *testing.T) {
	t.Setenv("NETFLIX_CONTAINER_NAME", "app")
	t.Setenv("TITUS_CONTAINER_NAME", "main")

	if got := tagsFromEnvVars()["nf.container"]; got != "app" {
		t.Errorf("Expected 'app', got '%s'", got)
	}
}

func TestTagsFromEnvVarsFallsBackToTitusContainerName(t *testing.T) {
	for _, value := range []string{"", " \t "} {
		t.Setenv("NETFLIX_CONTAINER_NAME", value)
		t.Setenv("TITUS_CONTAINER_NAME", "main")

		if got := tagsFromEnvVars()["nf.container"]; got != "main" {
			t.Errorf("NETFLIX_CONTAINER_NAME=%q: expected 'main', got '%s'", value, got)
		}
	}
}

func TestTagsFromEnvVarsNetflixContainerNameOnly(t *testing.T) {
	t.Setenv("NETFLIX_CONTAINER_NAME", " app ")
	t.Setenv("TITUS_CONTAINER_NAME", "")

	if got := tagsFromEnvVars()["nf.container"]; got != "app" {
		t.Errorf("Expected 'app', got '%s'", got)
	}
}

func TestTagsFromEnvVarsSkipsEmptyValues(t *testing.T) {
	t.Setenv("NETFLIX_CONTAINER_NAME", "")
	t.Setenv("TITUS_CONTAINER_NAME", "")
	t.Setenv("NETFLIX_PROCESS_NAME", "")

	tags := tagsFromEnvVars()
	if _, ok := tags["nf.container"]; ok {
		t.Errorf("Expected nf.container not to be set")
	}
	if _, ok := tags["nf.process"]; ok {
		t.Errorf("Expected nf.process not to be set")
	}
}

func TestTagsFromEnvVarsNoneSet(t *testing.T) {
	for _, envVar := range []string{"NETFLIX_CONTAINER_NAME", "TITUS_CONTAINER_NAME", "NETFLIX_PROCESS_NAME"} {
		t.Setenv(envVar, "") // restores the host value after the test
		_ = os.Unsetenv(envVar)
	}

	if tags := tagsFromEnvVars(); len(tags) != 0 {
		t.Errorf("Expected no tags, got %v", tags)
	}
}

// clearEnvTags clears the environment variables read by tagsFromEnvVars until the end of the test, so values
// injected into the host environment do not change the expected tags.
func clearEnvTags(t *testing.T) {
	t.Helper()
	t.Setenv("NETFLIX_CONTAINER_NAME", "")
	t.Setenv("TITUS_CONTAINER_NAME", "")
	t.Setenv("NETFLIX_PROCESS_NAME", "")
}
