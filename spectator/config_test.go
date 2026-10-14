package spectator

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestConfigMergesCommonTagsWithEnvVariables(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "container_name")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "process_name")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")
	t.Setenv("NETFLIX_CONTAINER_NAME", "") // an injected value would otherwise beat TITUS_CONTAINER_NAME

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

// Test env variables win over passed in values
func TestConfigMergesCommonTagsWithEnvVariablesAndPassedInValues(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "container_name_via_env")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "process_name_by_env")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")
	t.Setenv("NETFLIX_CONTAINER_NAME", "")

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

// Test NETFLIX_CONTAINER_NAME wins over TITUS_CONTAINER_NAME, and env variables win over passed in values
func TestConfigNetflixContainerNameWinsOverTitusAndPassedInValues(t *testing.T) {
	t.Setenv("NETFLIX_CONTAINER_NAME", "web")
	t.Setenv("TITUS_CONTAINER_NAME", "main")
	t.Setenv("NETFLIX_PROCESS_NAME", "processA")

	tags := map[string]string{
		"nf.app":       "app",
		"nf.container": "passed_in_container",
		"nf.process":   "passed_in_process",
	}

	config, err := NewConfig("", tags, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedTags := map[string]string{
		"nf.app":       "app",
		"nf.container": "web",
		"nf.process":   "processA",
	}

	if !reflect.DeepEqual(expectedTags, config.extraCommonTags) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, config.extraCommonTags)
	}
}

// Test NewConfigWithoutEnvTags keeps the passed in values and ignores the env variables
func TestNewConfigWithoutEnvTags(t *testing.T) {
	t.Setenv("NETFLIX_CONTAINER_NAME", "web")
	t.Setenv("TITUS_CONTAINER_NAME", "main")
	t.Setenv("NETFLIX_PROCESS_NAME", "processA")

	tags := map[string]string{
		"nf.app":       "app",
		"nf.container": "passed_in_container",
	}

	config, err := NewConfigWithoutEnvTags("memory", tags, nil, 1024, time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// the passed in nf.container is kept, and no nf.process is added from the env variables
	expectedTags := map[string]string{
		"nf.app":       "app",
		"nf.container": "passed_in_container",
	}

	if !reflect.DeepEqual(expectedTags, config.extraCommonTags) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, config.extraCommonTags)
	}

	if config.location != "memory" || config.bufferSize != 1024 || config.flushInterval != time.Second {
		t.Errorf("Expected location 'memory', bufferSize 1024 and flushInterval 1s, got '%s', %d and %v",
			config.location, config.bufferSize, config.flushInterval)
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
	expected := "udp"
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
	t.Setenv("NETFLIX_CONTAINER_NAME", "")

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
	clearEnvTags(t)

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
