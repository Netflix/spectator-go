package spectator

import (
	"strconv"
	"strings"
	"time"
)

type LogEntry struct {
	registry *Registry
	start    int64
	id       *Id
}

func (entry *LogEntry) SetStatusCode(code int) {
	entry.id = entry.id.WithTag("http.status", strconv.Itoa(code))
}

func (entry *LogEntry) SetSuccess() {
	extraTags := map[string]string{
		"ipc.result": "success",
		"ipc.status": "success",
	}
	entry.id = entry.id.WithTags(extraTags)
}

func (entry *LogEntry) SetError(err string) {
	extraTags := map[string]string{
		"ipc.result": "failure",
		"ipc.status": err,
	}
	entry.id = entry.id.WithTags(extraTags)
}

func (entry *LogEntry) SetAttempt(attemptNumber int, final bool) {
	extraTags := map[string]string{
		"ipc.attempt":       attempt(attemptNumber),
		"ipc.attempt.final": strconv.FormatBool(final),
	}
	entry.id = entry.id.WithTags(extraTags)
}

func (entry *LogEntry) Log() {
	duration := entry.registry.Clock().Nanos() - entry.start
	r := entry.registry
	r.config.IpcTimerRecord(r, entry.id, time.Duration(duration))
}

func attempt(n int) string {
	switch n {
	case 0:
		return "initial"
	case 1:
		return "second"
	default:
		return "third_up"
	}
}

func pathFromUrl(url string) string {
	if url == "" {
		return "/"
	}

	protoEnd := strings.IndexByte(url, ':')
	if protoEnd < 0 {
		return url
	}

	protocolLen := len(url) - protoEnd
	if protocolLen < 3 {
		// does not have ://
		return url
	}

	// find the path, skipping over protocol://
	pathBegin := strings.IndexByte(url[protoEnd+3:], '/')
	if pathBegin < 0 {
		return "/"
	}
	pathBegin += protoEnd + 3

	// find the first character that ends the path, could be beginning of query params, matrix params, or
	// url fragment
	pathEnd := strings.IndexAny(url[pathBegin+1:], "?#;")
	if pathEnd < 0 {
		// no query component
		return url[pathBegin:]
	}
	pathEnd += pathBegin + 1

	return url[pathBegin:pathEnd]
}

func NewLogEntry(registry *Registry, method string, url string) *LogEntry {
	tags := map[string]string{
		"owner":        "spectator-go",
		"ipc.endpoint": pathFromUrl(url),
		"http.method":  method,
		"http.status":  "-1",
	}
	return &LogEntry{registry, registry.Clock().Nanos(),
		registry.NewId("ipc.client.call", tags)}
}
