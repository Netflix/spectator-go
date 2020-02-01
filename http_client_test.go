package spectator

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

var ok = map[string]string{
	"status": "ok",
}

var errJson = map[string]string{
	"status": "error",
}

var okMsg, _ = json.Marshal(ok)
var errMsg, _ = json.Marshal(errJson)

func myMeters(registry *Registry) []Meter {
	var myMeters []Meter
	for _, meter := range registry.Meters() {
		if !strings.HasPrefix(meter.MeterId().name, "spectator.") {
			myMeters = append(myMeters, meter)
		}
	}
	return myMeters
}

func TestHttpClient_PostJsonOk(t *testing.T) {
	var log Logger
	const StartTime = 1
	clock := &ManualClock{StartTime}
	publishHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error("Unable to read body", err)
		}
		bodyStr := string(body)
		if bodyStr != "42" {
			t.Error("Unexpected body in request:", bodyStr)
		}
		_, _ = w.Write(okMsg)

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Unexpected content-type: %s", contentType)
		}
		clock.SetNanos(StartTime + 1000)
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	config := makeConfig(serverUrl)
	registry := NewRegistryWithClock(config, clock)
	log = registry.GetLogger()
	client := NewHttpClient(registry, 100*time.Millisecond)

	resp, err := client.postJson(config.Uri, []byte("42"))
	if err != nil {
		t.Error("Unexpected error", err)
	}

	if resp.status != 200 {
		t.Error("Expected 200 response. Got", resp.status)
	}

	meters := myMeters(registry)
	if len(meters) != 1 {
		t.Fatal("Expected 1 meter, got", len(meters))
	}

	expectedTags := map[string]string{
		"owner":             "spectator-go",
		"http.method":       "POST",
		"http.status":       "200",
		"ipc.attempt":       "initial",
		"ipc.attempt.final": "true",
		"ipc.endpoint":      "/",
		"ipc.result":        "success",
		"ipc.status":        "success",
	}
	expectedId := NewId("ipc.client.call", expectedTags)
	gotMeter := meters[0]
	if expectedId.name != gotMeter.MeterId().name || !reflect.DeepEqual(expectedTags, gotMeter.MeterId().tags) {
		log.Errorf("Unexpected meter registered. Expecting %v. Got %v", expectedId, gotMeter.MeterId())
	}

	assertTimer(t, gotMeter.(*Timer), 1, 1000, 1000*1000.0, 1000)
}

func TestHttpClient_PostJsonTimeout(t *testing.T) {
	var log Logger
	const StartTime = 1
	const Timeout = 1 * time.Millisecond
	clock := &ManualClock{StartTime}
	publishHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clock.SetFromDuration(StartTime + Timeout + 1)
		time.Sleep(Timeout + time.Millisecond) // trigger timeout
		_ = r.Body.Close()
		_, _ = io.WriteString(w, "\"Should have timed out\"")
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	config := makeConfig(serverUrl)
	registry := NewRegistryWithClock(config, clock)
	log = registry.GetLogger()
	client := NewHttpClient(registry, Timeout)

	resp, err := client.postJson(config.Uri, []byte("42"))
	// 400 is our catch all for errors that are results of exceptions
	if err != nil && (&resp != nil && resp.status != -1) {
		t.Error("Expected -1 response due to timeout, with error set. Got", resp)
	}

	meters := myMeters(registry)
	if len(meters) != 1 {
		t.Fatal("Expected 1 meter, got", len(meters))
	}

	expectedTags := map[string]string{
		"owner":             "spectator-go",
		"http.method":       "POST",
		"http.status":       "-1",
		"ipc.attempt":       "initial",
		"ipc.attempt.final": "true",
		"ipc.endpoint":      "/",
		"ipc.result":        "failure",
		"ipc.status":        "timeout",
	}
	expectedId := NewId("ipc.client.call", expectedTags)
	gotMeter := meters[0]
	if expectedId.name != gotMeter.MeterId().name || !reflect.DeepEqual(expectedTags, gotMeter.MeterId().tags) {
		log.Errorf("Unexpected meter registered. Expecting %v. Got %v", expectedId, gotMeter.MeterId())
	}

	total := int64(Timeout + 1)
	totalSq := float64(total) * float64(total)
	assertTimer(t, gotMeter.(*Timer), 1, total, totalSq, total)
}

func TestHttpClient_PostJson503(t *testing.T) {
	const StartTime = 1
	clock := &ManualClock{StartTime}
	publishHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error("Unable to read body", err)
		}
		bodyStr := string(body)
		if bodyStr != "42" {
			t.Error("Unexpected body in request:", bodyStr)
		}
		w.WriteHeader(503)
		_, _ = w.Write(errMsg)

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Unexpected content-type: %s", contentType)
		}
		clock.SetNanos(StartTime + 1000)
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	config := makeConfig(serverUrl)
	registry := NewRegistryWithClock(config, clock)
	client := NewHttpClient(registry, 100*time.Millisecond)

	resp, err := client.postJson(config.Uri, []byte("42"))
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	if resp.status != 503 {
		t.Error("Expected 503 response. Got", resp.status)
	}

	meters := myMeters(registry)
	sort.Slice(meters, func(i, j int) bool {
		return meters[i].MeterId().Tags()["ipc.attempt"] < meters[j].MeterId().Tags()["ipc.attempt"]
	})
	if len(meters) != 3 {
		t.Fatal("Expected 1 meter, got", len(meters))
	}

	baseTags := map[string]string{
		"owner":        "spectator-go",
		"http.method":  "POST",
		"http.status":  "503",
		"ipc.endpoint": "/",
		"ipc.result":   "failure",
		"ipc.status":   "http-error",
	}
	initial := map[string]string{
		"ipc.attempt":       "initial",
		"ipc.attempt.final": "false",
	}
	second := map[string]string{
		"ipc.attempt":       "second",
		"ipc.attempt.final": "false",
	}
	final := map[string]string{
		"ipc.attempt":       "third_up",
		"ipc.attempt.final": "true",
	}
	extra := []map[string]string{initial, second, final}

	for i, m := range meters {
		if m.MeterId().Name() != "ipc.client.call" {
			t.Errorf("Expected ipc.client.call got %s (%v)", m.MeterId().Name(), m.MeterId())
		}
		assertTags(t, m.MeterId().Tags(), baseTags, extra[i])
	}
}

func assertTags(t *testing.T, actual, base, extra map[string]string) {
	expected := make(map[string]string)
	for k, v := range base {
		expected[k] = v
	}
	for k, v := range extra {
		expected[k] = v
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}
