package spectator

import (
	"os"
	"reflect"
	"testing"
)

func TestNewRegistryConfiguredBy(t *testing.T) {
	r, err := NewRegistryConfiguredBy("test_config.json")
	if err != nil {
		t.Fatal("Unable to get a registry", err)
	}

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

func TestNewRegistryConfiguredBy_ExtraKeysAreIgnored(t *testing.T) {
	r, err := NewRegistryConfiguredBy("test_config_extra_keys.json")
	if err != nil {
		t.Fatal("Unable to get a registry", err)
	}

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

func TestConfigMergesCommonTagsWithEnvVariables(t *testing.T) {
	_ = os.Setenv("TITUS_CONTAINER_NAME", "container_name")
	_ = os.Setenv("NETFLIX_PROCESS_NAME", "process_name")
	defer os.Unsetenv("TITUS_CONTAINER_NAME")
	defer os.Unsetenv("NETFLIX_PROCESS_NAME")

	tags := map[string]string{
		"nf.app":     "app",
		"nf.account": "1234",
	}

	r, _ := NewRegistry(&Config{
		CommonTags: tags,
	})

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
	r, _ := NewRegistry(&Config{
		CommonTags: tags,
	})

	meterId := r.NewId("test_id", nil)

	// check that meter id has the expected tags
	expectedTags := map[string]string{
		"nf.app":       "app",
		"nf.account":   "1234",
		"nf.container": "passed_in_container",
		"nf.process":   "passed_in_process",
	}

	if !reflect.DeepEqual(expectedTags, meterId.Tags()) {
		t.Errorf("Expected tags %#v, got %#v", expectedTags, meterId.Tags())
	}
}

func TestGetLocation_ConfigValue(t *testing.T) {
	cfg := &Config{
		Location: "memory",
	}
	result := cfg.GetLocation()
	expected := "memory"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestGetLocation_EnvValue(t *testing.T) {
	_ = os.Setenv("SPECTATOR_OUTPUT_LOCATION", "stdout")
	defer os.Unsetenv("SPECTATOR_OUTPUT_LOCATION")

	cfg := &Config{}
	result := cfg.GetLocation()
	expected := "stdout"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestGetLocation_DefaultValue(t *testing.T) {
	cfg := &Config{}
	result := cfg.GetLocation()
	expected := "udp://127.0.0.1:1234"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
