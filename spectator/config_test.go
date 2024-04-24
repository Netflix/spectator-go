package spectator

import (
	"github.com/Netflix/spectator-go/spectator/logger"
	"os"
	"reflect"
	"testing"
)

func TestNewRegistryConfiguredBy(t *testing.T) {
	r, err := NewRegistryConfiguredBy("test_config.json")
	if err != nil {
		t.Fatal("Unable to get a registry", err)
	}

	logger := logger.NewDefaultLogger()
	expectedConfig := Config{
		Location:   "",
		CommonTags: map[string]string{"nf.app": "app", "nf.account": "1234"},
		Log:        logger,
	}

	// Set the same logger so that we can compare the configs
	cfg := r.config
	cfg.Log = logger

	if !reflect.DeepEqual(&expectedConfig, cfg) {
		t.Errorf("Expected config %#v, got %#v", expectedConfig, cfg)
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

	logger := logger.NewDefaultLogger()
	expectedConfig := Config{
		CommonTags: map[string]string{
			"nf.app":       "app",
			"nf.account":   "1234",
			"nf.container": "container_name",
			"nf.process":   "process_name",
		},
		Log: logger,
	}

	// Set the same logger so that we can compare the configs
	cfg := r.config
	cfg.Log = logger

	if !reflect.DeepEqual(&expectedConfig, cfg) {
		t.Errorf("Expected config %#v, got %#v", expectedConfig, cfg)
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
