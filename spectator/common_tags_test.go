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

	tags := tagsFromEnvVars()

	if tags["nf.container"] != "container_name" {
		t.Errorf("Expected 'container_name', got '%s'", tags["nf.container"])
	}

	if tags["nf.process"] != "process_name" {
		t.Errorf("Expected 'process_name', got '%s'", tags["nf.process"])
	}
}
