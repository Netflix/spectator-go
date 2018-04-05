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
	clock := &ManualClock{0, StartTime}
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
		clock.monotonic = StartTime + 1000
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	config := makeConfig(serverUrl)
	registry := NewRegistry(config)
	registry.clock = clock
	log = registry.log
	client := NewHttpClient(registry, 100*time.Millisecond)

	statusCode := client.PostJson(config.Uri, []byte("42"))
	if statusCode != 200 {
		t.Error("Expected 200 response. Got", statusCode)
	}

	meters := registry.Meters()
	if len(meters) != 1 {
		t.Fatal("Expected 1 meter, got", len(meters))
	}

	expectedTags := map[string]string{
		"status":     "2xx",
		"statusCode": "200",
		"client":     "spectator-go",
		"method":     "POST",
		"mode":       "http-client",
	}
	expectedId := newId("http.req.complete", expectedTags)
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
	clock := &ManualClock{0, StartTime}
	publishHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clock.monotonic = StartTime + Timeout + 1
		time.Sleep(Timeout + time.Millisecond) // trigger timeout
		r.Body.Close()
		io.WriteString(w, "\"Should have timed out\"")
	})

	server := httptest.NewServer(publishHandler)
	defer server.Close()

	serverUrl := server.URL

	config := makeConfig(serverUrl)
	registry := NewRegistry(config)
	registry.clock = clock
	log = registry.log
	client := NewHttpClient(registry, Timeout)

	statusCode := client.PostJson(config.Uri, []byte("42"))
	// 400 is our catch all for errors that are results of exceptions
	if statusCode != 400 {
		t.Error("Expected 200 response. Got", statusCode)
	}

	meters := registry.Meters()
	if len(meters) != 1 {
		t.Fatal("Expected 1 meter, got", len(meters))
	}

	expectedTags := map[string]string{
		"status":     "timeout",
		"statusCode": "timeout",
		"client":     "spectator-go",
		"method":     "POST",
		"mode":       "http-client",
	}
	expectedId := newId("http.req.complete", expectedTags)
	gotMeter := meters[0]
	if expectedId.name != gotMeter.MeterId().name || !reflect.DeepEqual(expectedTags, gotMeter.MeterId().tags) {
		log.Errorf("Unexpected meter registered. Expecting %v. Got %v", expectedId, gotMeter.MeterId())
	}

	total := int64(Timeout + 1)
	totalSq := float64(total) * float64(total)
	assertTimer(t, gotMeter.(*Timer), 1, total, totalSq, total)
}
