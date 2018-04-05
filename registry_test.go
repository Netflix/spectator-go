package spectator

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func makeConfig(uri string) *Config {
	return &Config{10 * time.Millisecond, 1 * time.Second, uri,
		map[string]string{
			"nf.app":     "test",
			"nf.cluster": "test-main",
			"nf.asg":     "test-main-v001",
			"nf.region":  "us-west-1",
		}}
}

var config = makeConfig("http://localhost:8080/")

func TestRegistry_Counter(t *testing.T) {
	r := NewRegistry(config)
	r.Counter("foo", nil).Increment()
	if v := r.Counter("foo", nil).Count(); v != 1 {
		t.Error("Counter needs to return a previously registered counter. Expected 1, got", v)
	}
}

func TestRegistry_DistributionSummary(t *testing.T) {
	r := NewRegistry(config)
	r.DistributionSummary("ds", nil).Record(100)
	if v := r.DistributionSummary("ds", nil).Count(); v != 1 {
		t.Error("DistributionSummary needs to return a previously registered meter. Expected 1, got", v)
	}
	if v := r.DistributionSummary("ds", nil).TotalAmount(); v != 100 {
		t.Error("Expected 100, Got", v)
	}
}

func TestRegistry_Gauge(t *testing.T) {
	r := NewRegistry(config)
	r.Gauge("g", nil).Set(100)
	if v := r.Gauge("g", nil).Get(); v != 100 {
		t.Error("Gauge needs to return a previously registered meter. Expected 100, got", v)
	}
}

func TestRegistry_Timer(t *testing.T) {
	r := NewRegistry(config)
	r.Timer("t", nil).Record(100)
	if v := r.Timer("t", nil).Count(); v != 1 {
		t.Error("Timer needs to return a previously registered meter. Expected 1, got", v)
	}
	if v := r.Timer("t", nil).TotalTime(); v != 100 {
		t.Error("Expected 100, Got", v)
	}
}

func TestRegistry_Start(t *testing.T) {
	r := NewRegistry(config)
	clock := &ManualClock{1, 1}
	r.clock = clock
	r.Counter("foo", nil).Increment()
	r.Start()
	time.Sleep(30 * time.Millisecond)
	r.Stop()
}

func TestRegistry_publish(t *testing.T) {
	const StartTime = 1
	clock := &ManualClock{StartTime, StartTime}
	publishHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Unexpected content-type: %s", contentType)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal("Unable to read body", err)
		}
		var payload []interface{}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			t.Fatal("Unable to unmarshal payload", err)
		}
		expected := []interface{}{
			// string table
			11.0, "count", "name", "nf.app", "nf.asg", "nf.cluster", "nf.region", "statistic", "test", "test-main", "test-main-v001", "us-west-1",
			// one measurement: a counter with value 10
			6.0, 2.0, 7.0, 4.0, 8.0, 3.0, 9.0, 5.0, 10.0, 6.0, 0.0, 1.0, 0.0,
			// op is 0 = add
			0.0,
			// delta is 10
			10.0}

		if !reflect.DeepEqual(expected, payload) {
			t.Errorf("Expected payload %v, got %v", expected, payload)
		}

		w.Write(okMsg)

		clock.monotonic = StartTime + 1000
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	cfg := makeConfig(serverUrl)
	r := NewRegistry(cfg)
	r.clock = clock

	r.Counter("foo", nil).Add(10)
	r.publish()
}
