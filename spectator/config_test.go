package spectator

import (
	"os"
	"reflect"
	"testing"
)

func TestConfigMergesCommonTagsWithEnvVariables(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "container_name")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "process_name")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")

	tags := map[string]string{
		"nf.app":     "app",
		"nf.account": "1234",
	}

	config, err := NewConfig("", tags, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	r, _ := NewRegistry(config)

	meterId := r.NewId("test_id", nil)

	// check that meter id has the expected tags
	expectedTags := map[string]string{
		"nf.app":       "app",
		"nf.account":   "1234",
		"nf.container": "container_name",
		"nf.process":   "process_name",
	}

	if !reflect.DeepEqual(expectedTags, meterId.Tags()) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, meterId.Tags())
	}
}

// Test passed in values wins over env variables
func TestConfigMergesCommonTagsWithEnvVariablesAndPassedInValues(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "container_name_via_env")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "process_name_by_env")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")

	tags := map[string]string{
		"nf.app":       "app",
		"nf.account":   "1234",
		"nf.container": "passed_in_container",
		"nf.process":   "passed_in_process",
	}

	config, err := NewConfig("", tags, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	r, _ := NewRegistry(config)

	meterId := r.NewId("test_id", nil)

	// check that meter id has the expected tags
	expectedTags := map[string]string{
		"nf.app":       "app",
		"nf.account":   "1234",
		"nf.container": "container_name_via_env",
		"nf.process":   "process_name_by_env",
	}

	if !reflect.DeepEqual(expectedTags, meterId.Tags()) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, meterId.Tags())
	}
}

func TestGetLocation_ConfigValue(t *testing.T) {
	cfg, err := NewConfig("memory", nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	result := cfg.location
	expected := "memory"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestGetLocation_EnvValue(t *testing.T) {
	_ = os.Setenv("SPECTATOR_OUTPUT_LOCATION", "stdout")
	defer os.Unsetenv("SPECTATOR_OUTPUT_LOCATION")

	cfg, err := NewConfig("", nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	result := cfg.location
	expected := "stdout"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestGetLocation_DefaultValue(t *testing.T) {
	cfg, err := NewConfig("", nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	result := cfg.location
	expected := "udp://127.0.0.1:1234"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// NewConfigShouldReturnErrorForInvalidLocation tests that NewConfig returns an error when provided an invalid location.
func TestNewConfigShouldReturnErrorForInvalidLocation(t *testing.T) {
	_, err := NewConfig("invalid_location", nil, nil)
	if err == nil {
		t.Errorf("Expected error for invalid location, got nil")
	}
}

func TestConfigMergesIgnoresCommonTagsWithEnvVariablesEmptyValues(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")

	tags := map[string]string{
		"nf.app":     "app",
		"nf.account": "1234",
	}

	config, err := NewConfig("", tags, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	r, _ := NewRegistry(config)

	meterId := r.NewId("test_id", nil)

	// check that meter id has the expected tags
	expectedTags := map[string]string{
		"nf.app":     "app",
		"nf.account": "1234",
	}

	if !reflect.DeepEqual(expectedTags, meterId.Tags()) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, meterId.Tags())
	}
}

func TestConfigMergesIgnoresTagsWithPassedInValuesEmptyValues(t *testing.T) {
	tags := map[string]string{
		"nf.app":     "app",
		"nf.account": "",
	}

	config, err := NewConfig("", tags, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	r, _ := NewRegistry(config)
	meterId := r.NewId("test_id", nil)

	// check that meter id has the expected tags
	expectedTags := map[string]string{
		"nf.app": "app",
	}

	if !reflect.DeepEqual(expectedTags, meterId.Tags()) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, meterId.Tags())
	}
}
