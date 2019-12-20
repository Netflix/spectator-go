package spectator

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

var ok = map[string]string{
	"status": "ok",
}

var okMsg, _ = json.Marshal(ok)

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
		w.Write(okMsg)

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

	statusCode, err := client.PostJson(config.Uri, []byte("42"))
	if err != nil {
		t.Error("Unexpected error", err)
	}

	if statusCode != 200 {
		t.Error("Expected 200 response. Got", statusCode)
	}

	meters := registry.Meters()
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
	expectedId := newId("ipc.client.call", expectedTags)
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
		r.Body.Close()
		io.WriteString(w, "\"Should have timed out\"")
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	config := makeConfig(serverUrl)
	registry := NewRegistryWithClock(config, clock)
	log = registry.GetLogger()
	client := NewHttpClient(registry, Timeout)

	statusCode, err := client.PostJson(config.Uri, []byte("42"))
	// 400 is our catch all for errors that are results of exceptions
	if statusCode != 400 || err == nil {
		t.Error("Expected 400 response due to timeout, with error set. Got", statusCode)
	}

	meters := registry.Meters()
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
	expectedId := newId("ipc.client.call", expectedTags)
	gotMeter := meters[0]
	if expectedId.name != gotMeter.MeterId().name || !reflect.DeepEqual(expectedTags, gotMeter.MeterId().tags) {
		log.Errorf("Unexpected meter registered. Expecting %v. Got %v", expectedId, gotMeter.MeterId())
	}

	total := int64(Timeout + 1)
	totalSq := float64(total) * float64(total)
	assertTimer(t, gotMeter.(*Timer), 1, total, totalSq, total)
}
