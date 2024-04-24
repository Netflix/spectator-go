// Package spectator provides a minimal Go implementation of the Netflix Java
// Spectator library. The goal of this package is to allow Go programs to emit
// metrics to Atlas.
//
// Please refer to the Java Spectator documentation for information on
// spectator / Atlas fundamentals: https://netflix.github.io/spectator/en/latest/
package spectator

import (
	"encoding/json"
	"github.com/Netflix/spectator-go/spectator/logger"
	"github.com/Netflix/spectator-go/spectator/meter"
	"github.com/Netflix/spectator-go/spectator/writer"
	"io/ioutil"
	"path/filepath"
)

// Meter represents the functionality presented by the individual meter types.
type Meter interface {
	MeterId() *meter.Id
}

type RegistryInterface interface {
	GetLogger() logger.Logger
	SetLogger(logger logger.Logger)
	NewMeter(id *meter.Id, meterFactory MeterFactoryFun) Meter
	NewId(name string, tags map[string]string) *meter.Id
	Counter(name string, tags map[string]string) *meter.Counter
	CounterWithId(id *meter.Id) *meter.Counter
	Timer(name string, tags map[string]string) *meter.Timer
	TimerWithId(id *meter.Id) *meter.Timer
	Gauge(name string, tags map[string]string) *meter.Gauge
	GaugeWithId(id *meter.Id) *meter.Gauge
	MaxGauge(name string, tags map[string]string) *meter.MaxGauge
	MaxGaugeWithId(id *meter.Id) *meter.MaxGauge
	DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary
	DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary
	PercentileDistributionSummary(name string, tags map[string]string) *meter.PercentileDistributionSummary
	PercentileDistributionSummaryWithId(id *meter.Id) *meter.PercentileDistributionSummary
	PercentileTimer(name string, tags map[string]string) *meter.PercentileTimer
	PercentileTimerWithId(id *meter.Id) *meter.PercentileTimer
	Close()
}

// MeterFactoryFun is a type to allow dependency injection of the function used to generate meters.
type MeterFactoryFun func() Meter

// Used to validate that Registry implements RegistryInterface at build time.
var _ RegistryInterface = (*Registry)(nil)

// Registry is the collection of meters being reported.
type Registry struct {
	config *Config
	writer writer.Writer
	quit   chan struct{}
}

// NewRegistryConfiguredBy loads a new Config JSON file from disk at the path specified.
func NewRegistryConfiguredBy(filePath string) (*Registry, error) {
	path := filepath.Clean(filePath)
	/* #nosec G304 */
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	registry, err := NewRegistry(&config)
	if err != nil {
		return nil, err
	}

	return registry, nil
}

// NewRegistry generates a new registry from the config.
//
// If config.IsEnabled is unset, it defaults to an implementation that returns
// true.
//
// If config.Log is unset, it defaults to using the default logger.
func NewRegistry(config *Config) (*Registry, error) {
	if config.Log == nil {
		config.Log = logger.NewDefaultLogger()
	}

	mergedTags := tagsFromEnvVars()
	// merge env var tags with config tags
	for k, v := range config.CommonTags {
		mergedTags[k] = v
	}
	config.CommonTags = mergedTags

	udpWriter, err := writer.NewWriter(config.GetLocation(), config.Log)
	if err != nil {
		return nil, err
	}

	r := &Registry{
		config: config,
		writer: udpWriter,
		quit:   make(chan struct{}),
	}

	return r, nil
}

// GetLogger returns the internal logger.
func (r *Registry) GetLogger() logger.Logger {
	return r.config.Log
}

func (r *Registry) SetLogger(logger logger.Logger) {
	r.config.Log = logger
}

// TODO investigate how to provide the same functionality for MeterFactory. Do we need to expose the writer?
// NewMeter creates a new Meter using the provided meterFactory
func (r *Registry) NewMeter(id *meter.Id, meterFactory MeterFactoryFun) Meter {
	return meterFactory()
}

// NewId calls meters.NewId() and adds the CommonTags registered in the config.
func (r *Registry) NewId(name string, tags map[string]string) *meter.Id {
	newId := meter.NewId(name, tags)

	if r.config.CommonTags != nil && len(r.config.CommonTags) > 0 {
		newId = newId.WithTags(r.config.CommonTags)
	}

	return newId
}

// Counter calls NewId() with the name and tags, and then calls r.CounterWithId()
// using that *Id.
func (r *Registry) Counter(name string, tags map[string]string) *meter.Counter {
	return meter.NewCounter(r.NewId(name, tags), r.writer)
}

func (r *Registry) CounterWithId(id *meter.Id) *meter.Counter {
	return meter.NewCounter(id, r.writer)
}

// Timer calls NewId() with the name and tags, and then calls r.TimerWithId() using that *Id.
func (r *Registry) Timer(name string, tags map[string]string) *meter.Timer {
	return meter.NewTimer(r.NewId(name, tags), r.writer)
}

// TimerWithId returns a new *Timer, using the provided meter identifier.
func (r *Registry) TimerWithId(id *meter.Id) *meter.Timer {
	return meter.NewTimer(id, r.writer)
}

func (r *Registry) Gauge(name string, tags map[string]string) *meter.Gauge {
	return meter.NewGauge(r.NewId(name, tags), r.writer)
}

func (r *Registry) GaugeWithId(id *meter.Id) *meter.Gauge {
	return meter.NewGauge(id, r.writer)
}

func (r *Registry) MaxGauge(name string, tags map[string]string) *meter.MaxGauge {
	return meter.NewMaxGauge(r.NewId(name, tags), r.writer)
}

func (r *Registry) MaxGaugeWithId(id *meter.Id) *meter.MaxGauge {
	return meter.NewMaxGauge(id, r.writer)
}

func (r *Registry) DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary {
	return meter.NewDistributionSummary(r.NewId(name, tags), r.writer)
}

func (r *Registry) DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary {
	return meter.NewDistributionSummary(id, r.writer)
}

func (r *Registry) PercentileDistributionSummary(name string, tags map[string]string) *meter.PercentileDistributionSummary {
	return meter.NewPercentileDistributionSummary(r.NewId(name, tags), r.writer)
}

func (r *Registry) PercentileDistributionSummaryWithId(id *meter.Id) *meter.PercentileDistributionSummary {
	return meter.NewPercentileDistributionSummary(id, r.writer)
}

func (r *Registry) PercentileTimer(name string, tags map[string]string) *meter.PercentileTimer {
	return meter.NewPercentileTimer(r.NewId(name, tags), r.writer)
}

func (r *Registry) PercentileTimerWithId(id *meter.Id) *meter.PercentileTimer {
	return meter.NewPercentileTimer(id, r.writer)
}

func (r *Registry) Close() {
	err := r.writer.Close()

	if err != nil {
		r.GetLogger().Errorf("Error closing writer: %v", err)
	}
}
