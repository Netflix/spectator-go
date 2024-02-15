package spectator

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func measurementsToMap(ms []Measurement) map[string]float64 {
	var result = make(map[string]float64)
	for _, m := range ms {
		idStr := fmt.Sprintf("%s|%s", m.Id().Name(), m.Id().Tags()["statistic"])
		p := m.Id().Tags()["percentile"]
		if p != "" {
			idStr += "|" + p
		}
		id := m.id.tags["id"]
		if id != "" {
			idStr += "|" + id
		}
		result[idStr] = m.Value()
	}
	return result
}

func TestLogEntry_Log(t *testing.T) {
	clock := ManualClock{}
	clock.SetFromDuration(0)
	entry := getLogEntryWithClock(&Config{}, &clock)

	entry.SetAttempt(0, true)
	entry.SetStatusCode(200)
	entry.SetSuccess()
	clock.SetFromDuration(500 * time.Millisecond)
	entry.Log()

	ms := entry.registry.Measurements()
	actual := measurementsToMap(ms)
	expected := map[string]float64{
		"ipc.client.call|totalTime":      0.5,
		"ipc.client.call|totalOfSquares": 0.25,
		"ipc.client.call|max":            0.5,
		"ipc.client.call|count":          1,

		"spectator.registrySize|count":          1,
		"spectator.registrySize|max":            6,
		"spectator.registrySize|totalAmount":    6,
		"spectator.registrySize|totalOfSquares": 36,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v\nGot %v", expected, actual)
	}
}

func TestLogEntry_LogCustom(t *testing.T) {
	clock := ManualClock{}
	clock.SetFromDuration(0)
	config := Config{}
	config.IpcTimerRecord = func(registry *Registry, id *Id, duration time.Duration) {
		registry.TimerWithId(id).Record(duration)
		registry.CounterWithId(id.WithStat("percentile").WithTag("percentile", "T007D")).Increment()
	}
	entry := getLogEntryWithClock(&config, &clock)

	entry.SetAttempt(0, true)
	entry.SetStatusCode(200)
	entry.SetSuccess()
	clock.SetFromDuration(500 * time.Millisecond)
	entry.Log()

	ms := entry.registry.Measurements()
	actual := measurementsToMap(ms)
	expected := map[string]float64{
		"ipc.client.call|totalTime":        0.5,
		"ipc.client.call|totalOfSquares":   0.25,
		"ipc.client.call|max":              0.5,
		"ipc.client.call|count":            1,
		"ipc.client.call|percentile|T007D": 1,

		"spectator.registrySize|count":          1,
		"spectator.registrySize|max":            7,
		"spectator.registrySize|totalAmount":    7,
		"spectator.registrySize|totalOfSquares": 49,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected %v\nGot %v", expected, actual)
	}
}

func TestLogEntry_SetSuccess(t *testing.T) {
	entry := getLogEntry()
	entry.SetSuccess()
	if v := entry.id.Tags()["ipc.result"]; v != "success" {
		t.Error("Expected success, got", v)
	}

	if v := entry.id.Tags()["ipc.status"]; v != "success" {
		t.Error("Expected success, got", v)
	}
}

func TestLogEntry_SetError(t *testing.T) {
	entry := getLogEntry()
	entry.SetError("some-err")
	if v := entry.id.Tags()["ipc.result"]; v != "failure" {
		t.Error("Expected failure, got", v)
	}

	if v := entry.id.Tags()["ipc.status"]; v != "some-err" {
		t.Error("Expected some-err, got", v)
	}
}

func TestLogEntry_SetStatusCode(t *testing.T) {
	entry := getLogEntry()
	entry.SetStatusCode(404)
	if c := entry.id.Tags()["http.status"]; c != "404" {
		t.Error("Expected 404, got", c)
	}
}

func TestLogEntry_SetAttempt(t *testing.T) {
	entry := getLogEntry()
	entry.SetAttempt(0, true)
	if attempt := entry.id.Tags()["ipc.attempt"]; attempt != "initial" {
		t.Error("Expected initial, got", attempt)
	}
	if final := entry.id.Tags()["ipc.attempt.final"]; final != "true" {
		t.Error("Expected true, got", final)
	}
	entry.SetAttempt(1, false)
	if attempt := entry.id.Tags()["ipc.attempt"]; attempt != "second" {
		t.Error("Expected second, got", attempt)
	}
	if final := entry.id.Tags()["ipc.attempt.final"]; final != "false" {
		t.Error("Expected false, got", final)
	}
	entry.SetAttempt(10, true)
	if attempt := entry.id.Tags()["ipc.attempt"]; attempt != "third_up" {
		t.Error("Expected third_up, got", attempt)
	}
	if final := entry.id.Tags()["ipc.attempt.final"]; final != "true" {
		t.Error("Expected true, got", final)
	}
}

func getLogEntry() *LogEntry {
	return getLogEntryWithClock(&Config{}, &SystemClock{})
}

func getLogEntryWithClock(config *Config, clock Clock) *LogEntry {
	registry := NewRegistryWithClock(config, clock)
	entry := NewLogEntry(registry, "POST", "/api/v4/update")
	return entry
}

func TestPathFromUrl_Empty(t *testing.T) {
	if p := pathFromUrl(""); p != "/" {
		t.Error("Empty Url should get / - Got", p)
	}
}

func TestPathFromUrl_OnlyPath(t *testing.T) {
	if p := pathFromUrl("/fo"); p != "/fo" {
		t.Error("Just a path should get the same value back - Got", p)
	}
}

func TestPathFromUrl_NoSlashSlash(t *testing.T) {
	if p := pathFromUrl("foo:/"); p != "foo:/" {
		t.Error("No protocol:// - Got", p)
	}
}

func TestPathFromUrl_OnlyHost(t *testing.T) {
	if p := pathFromUrl("ftp://foo.example.com"); p != "/" {
		t.Error("Only host - Got", p)
	}
}

func TestPathFromUrl_NoQueryString(t *testing.T) {
	p := pathFromUrl("ftp://foo.example.com/api/v4/update")
	if p != "/api/v4/update" {
		t.Error("No query string - Got", p)
	}

	p = pathFromUrl("ftp://foo.example.com/api/v4/update?")
	if p != "/api/v4/update" {
		t.Error("No query string - Got", p)
	}
}

func TestPathFromUrl_QueryString(t *testing.T) {
	p := pathFromUrl("ftp://foo.example.com/api/v4/update?foo=bar&baz=/foo")
	if p != "/api/v4/update" {
		t.Error("Url with query string - Got", p)
	}
}

func TestPathFromUrl_MatrixFrag(t *testing.T) {
	p := pathFromUrl("ftp://foo.example.com/api/v4/update;foo=bar#someAnchor")
	if p != "/api/v4/update" {
		t.Error("Url with query string - Got", p)
	}
}
