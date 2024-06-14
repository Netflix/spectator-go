// Package spectator provides a minimal Go implementation of the Netflix Java
// Spectator library. The goal of this package is to allow Go programs to emit
// metrics to Atlas.
//
// Please refer to the Java Spectator documentation for information on
// spectator / Atlas fundamentals: https://netflix.github.io/spectator/en/latest/
package spectator

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"github.com/Netflix/spectator-go/v2/spectator/meter"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"time"
)

// Meter represents the functionality presented by the individual meter types.
type Meter interface {
	MeterId() *meter.Id
}

// Registry is the main entry point for interacting with the Spectator library.
type Registry interface {
	GetLogger() logger.Logger
	NewId(name string, tags map[string]string) *meter.Id
	AgeGauge(name string, tags map[string]string) *meter.AgeGauge
	AgeGaugeWithId(id *meter.Id) *meter.AgeGauge
	Counter(name string, tags map[string]string) *meter.Counter
	CounterWithId(id *meter.Id) *meter.Counter
	DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary
	DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary
	Gauge(name string, tags map[string]string) *meter.Gauge
	GaugeWithId(id *meter.Id) *meter.Gauge
	GaugeWithTTL(name string, tags map[string]string, ttl time.Duration) *meter.Gauge
	GaugeWithIdWithTTL(id *meter.Id, ttl time.Duration) *meter.Gauge
	MaxGauge(name string, tags map[string]string) *meter.MaxGauge
	MaxGaugeWithId(id *meter.Id) *meter.MaxGauge
	MonotonicCounter(name string, tags map[string]string) *meter.MonotonicCounter
	MonotonicCounterWithId(id *meter.Id) *meter.MonotonicCounter
	MonotonicCounterUint(name string, tags map[string]string) *meter.MonotonicCounterUint
	MonotonicCounterUintWithId(id *meter.Id) *meter.MonotonicCounterUint
	PercentileDistributionSummary(name string, tags map[string]string) *meter.PercentileDistributionSummary
	PercentileDistributionSummaryWithId(id *meter.Id) *meter.PercentileDistributionSummary
	PercentileTimer(name string, tags map[string]string) *meter.PercentileTimer
	PercentileTimerWithId(id *meter.Id) *meter.PercentileTimer
	Timer(name string, tags map[string]string) *meter.Timer
	TimerWithId(id *meter.Id) *meter.Timer
	GetWriter() writer.Writer
	Close()
}

// Used to validate that spectatordRegistry implements Registry at build time.
var _ Registry = (*spectatordRegistry)(nil)

type spectatordRegistry struct {
	config *Config
	writer writer.Writer
	logger logger.Logger
}

// NewRegistry generates a new registry from a passed Config created through NewConfig.
func NewRegistry(config *Config) (Registry, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.location == "" {
		// Config was not created using NewConfig. Set a default config instead of using the passed one
		config, _ = NewConfig("", nil, nil)
	}

	newWriter, err := writer.NewWriter(config.location, config.log)
	if err != nil {
		return nil, err
	}

	r := &spectatordRegistry{
		config: config,
		writer: newWriter,
		logger: config.log,
	}

	return r, nil
}

// GetLogger returns the internal logger.
func (r *spectatordRegistry) GetLogger() logger.Logger {
	return r.logger
}

// NewId calls meters.NewId() and adds the commonTags registered in the config.
func (r *spectatordRegistry) NewId(name string, tags map[string]string) *meter.Id {
	newId := meter.NewId(name, tags)

	if r.config.commonTags != nil && len(r.config.commonTags) > 0 {
		newId = newId.WithTags(r.config.commonTags)
	}

	return newId
}

func (r *spectatordRegistry) AgeGauge(name string, tags map[string]string) *meter.AgeGauge {
	return meter.NewAgeGauge(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) AgeGaugeWithId(id *meter.Id) *meter.AgeGauge {
	return meter.NewAgeGauge(id, r.writer)
}

func (r *spectatordRegistry) Counter(name string, tags map[string]string) *meter.Counter {
	return meter.NewCounter(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) CounterWithId(id *meter.Id) *meter.Counter {
	return meter.NewCounter(id, r.writer)
}

func (r *spectatordRegistry) DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary {
	return meter.NewDistributionSummary(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary {
	return meter.NewDistributionSummary(id, r.writer)
}

func (r *spectatordRegistry) Gauge(name string, tags map[string]string) *meter.Gauge {
	return meter.NewGauge(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) GaugeWithId(id *meter.Id) *meter.Gauge {
	return meter.NewGauge(id, r.writer)
}

func (r *spectatordRegistry) GaugeWithTTL(name string, tags map[string]string, duration time.Duration) *meter.Gauge {
	return meter.NewGaugeWithTTL(r.NewId(name, tags), r.writer, duration)
}

func (r *spectatordRegistry) GaugeWithIdWithTTL(id *meter.Id, duration time.Duration) *meter.Gauge {
	return meter.NewGaugeWithTTL(id, r.writer, duration)
}

func (r *spectatordRegistry) MaxGauge(name string, tags map[string]string) *meter.MaxGauge {
	return meter.NewMaxGauge(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) MaxGaugeWithId(id *meter.Id) *meter.MaxGauge {
	return meter.NewMaxGauge(id, r.writer)
}

func (r *spectatordRegistry) MonotonicCounter(name string, tags map[string]string) *meter.MonotonicCounter {
	return meter.NewMonotonicCounter(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) MonotonicCounterWithId(id *meter.Id) *meter.MonotonicCounter {
	return meter.NewMonotonicCounter(id, r.writer)
}

func (r *spectatordRegistry) MonotonicCounterUint(name string, tags map[string]string) *meter.MonotonicCounterUint {
	return meter.NewMonotonicCounterUint(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) MonotonicCounterUintWithId(id *meter.Id) *meter.MonotonicCounterUint {
	return meter.NewMonotonicCounterUint(id, r.writer)
}

func (r *spectatordRegistry) PercentileDistributionSummary(name string, tags map[string]string) *meter.PercentileDistributionSummary {
	return meter.NewPercentileDistributionSummary(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) PercentileDistributionSummaryWithId(id *meter.Id) *meter.PercentileDistributionSummary {
	return meter.NewPercentileDistributionSummary(id, r.writer)
}

func (r *spectatordRegistry) PercentileTimer(name string, tags map[string]string) *meter.PercentileTimer {
	return meter.NewPercentileTimer(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) PercentileTimerWithId(id *meter.Id) *meter.PercentileTimer {
	return meter.NewPercentileTimer(id, r.writer)
}

func (r *spectatordRegistry) Timer(name string, tags map[string]string) *meter.Timer {
	return meter.NewTimer(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) TimerWithId(id *meter.Id) *meter.Timer {
	return meter.NewTimer(id, r.writer)
}

func (r *spectatordRegistry) GetWriter() writer.Writer {
	return r.writer
}

func (r *spectatordRegistry) Close() {
	err := r.writer.Close()

	if err != nil {
		r.GetLogger().Errorf("Error closing writer: %v", err)
	}
}
