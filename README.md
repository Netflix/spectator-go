[![Go Reference](https://pkg.go.dev/badge/github.com/Netflix/spectator-go.svg)](https://pkg.go.dev/github.com/Netflix/spectator-go)
[![Snapshot](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/snapshot.yml)
[![Release](https://github.com/Netflix/spectator-go/actions/workflows/release.yml/badge.svg)](https://github.com/Netflix/spectator-go/actions/workflows/release.yml)

# Spectator-go

> :warning: Experimental
 
Simple library for instrumenting code to record dimensional time series.

## Description

This implements a basic [Spectator](https://github.com/Netflix/spectator) library for instrumenting Go applications. 
It consists of a thin client designed to send metrics to [spectatord](https://github.com/Netflix-Skunkworks/spectatord).

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
