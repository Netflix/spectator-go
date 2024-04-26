[![Go Reference](https://pkg.go.dev/badge/github.com/Netflix/spectator-go.svg)](https://pkg.go.dev/github.com/Netflix/spectator-go)
[![Snapshot](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml)
[![Release](https://github.com/Netflix/spectator-go/actions/workflows/release.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/release.yml)

# Spectator-go

> :warning: Experimental

Simple library for instrumenting code to record dimensional time series.

## Description

This implements a basic [Spectator](https://github.com/Netflix/spectator) library for instrumenting Go applications.
It consists of a thin client designed to send metrics
through [spectatord](https://github.com/Netflix-Skunkworks/spectatord).

## Instrumenting Code

```go
package main

import (
	"github.com/Netflix/spectator-go/spectator"
	"github.com/Netflix/spectator-go/spectator/meter"
	"strconv"
	"time"
)

type Server struct {
	registry       spectator.Registry
	requestCountId *meter.Id
	requestLatency *meter.Timer
	responseSizes  *meter.DistributionSummary
}

type Request struct {
	country string
}

type Response struct {
	status int
	size   int64
}

func (s *Server) Handle(request *Request) (res *Response) {
	start := time.Now()

	// initialize response
	res = &Response{200, 64}

	// Update the counter with dimensions based on the request.
	tags := map[string]string{
		"country": request.country,
		"status":  strconv.Itoa(res.status),
	}
	requestCounterWithTags := s.requestCountId.WithTags(tags)
	counter := s.registry.CounterWithId(requestCounterWithTags)
	counter.Increment()

	// ...
	s.requestLatency.Record(time.Since(start))
	s.responseSizes.Record(res.size)
	return
}

func newServer(registry spectator.Registry) *Server {
	return &Server{
		registry,
		registry.NewId("server.requestCount", nil),
		registry.Timer("server.requestLatency", nil),
		registry.DistributionSummary("server.responseSizes", nil),
	}
}

func getNextRequest() *Request {
	// ...
	return &Request{"US"}
}

func main() {
	commonTags := map[string]string{"nf.app": "example", "nf.region": "us-west-1"}
	config := &spectator.Config{CommonTags: commonTags}

	registry, _ := spectator.NewRegistry(config)
	defer registry.Close()

	// optionally set custom logger (it must implement Debugf, Infof, Errorf)
	// registry.SetLogger(logger)

	server := newServer(*registry)

	for i := 1; i < 3; i++ {
		// get a request
		req := getNextRequest()
		server.Handle(req)
	}
}
```

## Logging

Logging is implemented with the standard Golang [slog package](https://pkg.go.dev/log/slog). The logger
defines interfaces for [Debugf, Infof, and Errorf](./spectator/logger/logger.go). There are useful messages
implemented at the Debug level which can help diagnose the metric publishing workflow.

---

## Migrating from 0.2.X to 0.3.X

Version 0.3 consists of a major rewrite that turns spectator-go into a thin client designed to send metrics through
[spectatord](https://github.com/Netflix-Skunkworks/spectatord). As a result some functionality has been moved to other
packages or removed.

### New

#### Writers

`spectator.Registry` now supports different writers. The default writer is `writer.UdpWriter` which sends metrics
to spectatord through UDP.

Writers can be configured through `spectator.Config.Location` (`sidecar.output-location` in file based configuration).
Possible values are:

- `none`
- `stdout`
- `stderr`
- `memory`
- `file:///path/to/file`
- `unix:///path/to/socket`
- `udp://host:port`

Location can also be set through the environment variable `SPECTATOR_OUTPUT_LOCATION`. If both are set, the Config value
takes precedence over the environment variable.

#### Meters

The following new Meters have been added:

- `meter.MaxGauge`
- `meter.Gauge` with TTL

#### Common Tags

Common tags are now automatically added to all Meters. Their values are read from the environment variables.

| Tag          | Environment Variable |
|--------------|----------------------|
| nf.container | TITUS_CONTAINER_NAME |
| nf.process   | NETFLIX_PROCESS_NAME |

### Moved

- Runtime metrics collection has been moved
  to [spectator-go-runtime-metrics](https://github.com/Netflix/spectator-go-runtime-metrics). Follow instructions in
  the [README](https://github.com/Netflix/spectator-go-runtime-metrics) to enable
  collection.
- Some types have been moved to different packages. For example, `spectator.Counter` is now in `meter.Counter`.

### Removed

- `spectator.HttpClient` has been removed. Use the standard `http.Client` instead.
- `spectator.Meter`s no longer has a `Measure() []Measurement` function. Meters are now stateless and do not store
  measurements.
- `spectator.Clock` has been removed. Use the standard `time` package instead.
- `spectator.Config` has been greatly simplified.
    - If you're using file based configuration, `"common_tags"` has been renamed to `"sidecar.common-tags"`.
- `spectator.Registry` no longer has a `Start()` function. It is now automatically started when created.
- `spectator.Registry` no longer has a `Stop()` function. Instead, use `Close()` to close the registry.
- `spectator.Config.IpcTimerRecord` has been removed. Use a `meter.Timer` instead to record Ipc metrics.
- `spectator.MeterFactoryFun` has been removed. If you need to create a custom meter you can do so by wrapping one of
  the meters returned by `spectator.Registry`.
- `spectator.Registry` no longer reports `spectator.measurements` metrics. Instead, you can use spectatord metrics to
  troubleshoot.
- `spectator.Registry` no longer keep track of the Meters it creates. This means that you can't get a list of all Meters
  from the Registry. If you need to keep track of Meters, you can do so in your application code.
- `Percentile*` meters no longer support defining min/max values.

### Migration steps

1. Make sure you're not relying on any of the [removed functionality](#removed).
2. Update imports to use `meters` package instead of `spectator` for Meters.
3. If you're using file-based configuration rename `"common_tags"` to `"sidecar.common-tags"`.
4. If you want to collect runtime metrics
   pull [spectator-go-runtime-metrics](https://github.com/Netflix/spectator-go-runtime-metrics) and follow the
   instructions in the
   [README](https://github.com/Netflix/spectator-go-runtime-metrics)
5. If you use `PercentileDistributionSummary` or `PercentileTimer` you need to update your code to use the respective
   functions provided by the `Registry to initialize these meters.