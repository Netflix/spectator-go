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

	logger := defaultLogger()
	expectedConfig := Config{
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
	r := NewRegistry(&Config{
		CommonTags: tags,
	})

	logger := defaultLogger()
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

//func TestRegistry_Counter(t *testing.T) {
//	r := NewRegistry(config)
//	r.Counter("foo", nil).Increment()
//	if v := r.Counter("foo", nil).Count(); v != 1 {
//		t.Error("Counter needs to return a previously registered counter. Expected 1, got", v)
//	}
//}
//
//func TestRegistry_DistributionSummary(t *testing.T) {
//	r := NewRegistry(config)
//	r.DistributionSummary("ds", nil).Record(100)
//	if v := r.DistributionSummary("ds", nil).Count(); v != 1 {
//		t.Error("DistributionSummary needs to return a previously registered meter. Expected 1, got", v)
//	}
//	if v := r.DistributionSummary("ds", nil).TotalAmount(); v != 100 {
//		t.Error("Expected 100, Got", v)
//	}
//}
//
//func TestRegistry_Gauge(t *testing.T) {
//	r := NewRegistry(config)
//	r.Gauge("g", nil).Set(100)
//	if v := r.Gauge("g", nil).Get(); v != 100 {
//		t.Error("Gauge needs to return a previously registered meter. Expected 100, got", v)
//	}
//}
//
//func TestRegistry_Timer(t *testing.T) {
//	r := NewRegistry(config)
//	r.Timer("t", nil).Record(100)
//	if v := r.Timer("t", nil).Count(); v != 1 {
//		t.Error("Timer needs to return a previously registered meter. Expected 1, got", v)
//	}
//	if v := r.Timer("t", nil).TotalTime(); v != 100 {
//		t.Error("Expected 100, Got", v)
//	}
//}
