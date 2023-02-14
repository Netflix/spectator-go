[![Go Reference](https://pkg.go.dev/badge/github.com/Netflix/spectator-go.svg)](https://pkg.go.dev/github.com/Netflix/spectator-go)
[![Snapshot](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml)
[![Release](https://github.com/Netflix/spectator-go/actions/workflows/release.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/release.yml)

# Spectator-go

> :warning: Experimental
 
Simple library for instrumenting code to record dimensional time series.

## Description

This implements a basic [Spectator](https://github.com/Netflix/spectator)
library for instrumenting golang applications, sending metrics to an Atlas
aggregator service.

## Instrumenting Code

```go
package main

import (
	"github.com/Netflix/spectator-go"
	"strconv"
	"time"
)

type Server struct {
	registry       *spectator.Registry
	requestCountId *spectator.Id
	requestLatency *spectator.Timer
	responseSizes  *spectator.DistributionSummary
}

type Request struct {
	country string
}

type Response struct {
	status int
	size   int64
}

func (s *Server) Handle(request *Request) (res *Response) {
	clock := s.registry.Clock()
	start := clock.Now()

	// initialize res
	res = &Response{200, 64}

	// Update the counter id with dimensions based on the request. The
	// counter will then be looked up in the registry which should be
	// fairly cheap, such as lookup of id object in a map
	// However, it is more expensive than having a local variable set
	// to the counter.
	cntId := s.requestCountId.WithTag("country", request.country).WithTag("status", strconv.Itoa(res.status))
	s.registry.CounterWithId(cntId).Increment()

	// ...
	s.requestLatency.Record(clock.Now().Sub(start))
	s.responseSizes.Record(res.size)
	return
}

func newServer(registry *spectator.Registry) *Server {
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
	config := &spectator.Config{Frequency: 5 * time.Second, Timeout: 1 * time.Second,
		Uri: "http://example.org/api/v1/publish", CommonTags: commonTags}
	registry := spectator.NewRegistry(config)

	// optionally set custom logger (it must implement Debugf, Infof, Errorf)
	// registry.SetLogger(logger)
	registry.Start()
	defer registry.Stop()

	// collect memory and file descriptor metrics
	spectator.CollectRuntimeMetrics(registry)

	server := newServer(registry)

	for i := 1; i < 3; i++ {
		// get a request
		req := getNextRequest()
		server.Handle(req)
	}
}
```

## Logging

Logging is implemented with the standard Golang [log package](https://pkg.go.dev/log). The logger
defines interfaces for [Debugf, Infof, and Errorf](./logger.go#L10-L14) which means that under
normal operation, you will see log messages for all of these levels. There are
[useful messages](https://github.com/Netflix/spectator-go/blob/master/registry.go#L268-L273)
implemented at the Debug level which can help diagnose the metric publishing workflow. If you do
not see any of these messages, then it is an indication that the Registry may not be started.

If you do not wish to see debug log messages from spectator-go, then you should configure a custom
logger which implements the Logger interface. A library such as [Zap](https://github.com/uber-go/zap)
can provide this functionality, which will then allow for log level control at the command line
with the `--log-level=debug` flag.

## Debugging Metric Payloads

Set the following environment variable to enumerate the metrics payloads which
are sent to the backend. This is useful for debugging metric publishing issues.

```
export SPECTATOR_DEBUG_PAYLOAD=1
```

## Known Issues

### Unable to close body: context canceled

If you see the following two error messages repeated in your application logs, then you are running
version 1.16.10 or 1.17.3 of Golang which introduced a regression ([Issue#49366]). See [PR#59] in
this project for related discussion.

```
level=error msg="Could not POST measurements: HTTP 200 context canceled"
level=error msg="Unable to close body: context canceled"
```

[Issue#49366]: https://github.com/golang/go/issues/49366
[PR#59]: https://github.com/Netflix/spectator-go/pull/59
