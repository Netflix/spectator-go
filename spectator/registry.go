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
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Meter represents the functionality presented by the individual meter types.
type Meter interface {
	MeterId() *meter.Id
}

// Registry is the main entry point for interacting with the Spectator library.
type Registry interface {
	GetLogger() logger.Logger
	SetLogger(logger logger.Logger)
	NewId(name string, tags map[string]string) *meter.Id
	Counter(name string, tags map[string]string) *meter.Counter
	CounterWithId(id *meter.Id) *meter.Counter
	MonotonicCounter(name string, tags map[string]string) *meter.MonotonicCounter
	MonotonicCounterWithId(id *meter.Id) *meter.MonotonicCounter
	Timer(name string, tags map[string]string) *meter.Timer
	TimerWithId(id *meter.Id) *meter.Timer
	Gauge(name string, tags map[string]string) *meter.Gauge
	GaugeWithId(id *meter.Id) *meter.Gauge
	GaugeWithTTL(name string, tags map[string]string, ttl time.Duration) *meter.Gauge
	GaugeWithIdWithTTL(id *meter.Id, ttl time.Duration) *meter.Gauge
	AgeGauge(name string, tags map[string]string) *meter.AgeGauge
	AgeGaugeWithId(id *meter.Id) *meter.AgeGauge
	MaxGauge(name string, tags map[string]string) *meter.MaxGauge
	MaxGaugeWithId(id *meter.Id) *meter.MaxGauge
	DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary
	DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary
	PercentileDistributionSummary(name string, tags map[string]string) *meter.PercentileDistributionSummary
	PercentileDistributionSummaryWithId(id *meter.Id) *meter.PercentileDistributionSummary
	PercentileTimer(name string, tags map[string]string) *meter.PercentileTimer
	PercentileTimerWithId(id *meter.Id) *meter.PercentileTimer
	GetWriter() writer.Writer
	Close()
}

// Used to validate that spectatordRegistry implements Registry at build time.
var _ Registry = (*spectatordRegistry)(nil)

type spectatordRegistry struct {
	config *Config
	writer writer.Writer
	logger      logger.Logger
	loggerMutex sync.RWMutex
}

// NewRegistryConfiguredBy loads a new Config JSON file from disk at the path specified.
func NewRegistryConfiguredBy(filePath string) (Registry, error) {
	path := filepath.Clean(filePath)
	/* #nosec G304 */
	bytes, err := os.ReadFile(path)
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
// If config.Log is unset, it defaults to using the default logger.
func NewRegistry(config *Config) (Registry, error) {
	log := config.Log
	if log == nil {
		log = logger.NewDefaultLogger()
	}

	mergedTags := tagsFromEnvVars()
	// merge env var tags with config tags
	for k, v := range config.CommonTags {
		mergedTags[k] = v
	}
	config.CommonTags = mergedTags

	newWriter, err := writer.NewWriter(config.GetLocation(), log)
	if err != nil {
		return nil, err
	}

	log.Infof("Initializing Registry using writer: %T", newWriter)

	r := &spectatordRegistry{
		config: config,
		writer: newWriter,
		logger: log,
	}

	return r, nil
}

// GetLogger returns the internal logger.
func (r *spectatordRegistry) GetLogger() logger.Logger {
	r.loggerMutex.RLock()
	defer r.loggerMutex.RUnlock()
	return r.logger
}

func (r *spectatordRegistry) SetLogger(logger logger.Logger) {
	r.loggerMutex.Lock()
	defer r.loggerMutex.Unlock()
	r.logger = logger
}

// NewId calls meters.NewId() and adds the CommonTags registered in the config.
func (r *spectatordRegistry) NewId(name string, tags map[string]string) *meter.Id {
	newId := meter.NewId(name, tags)

	if r.config.CommonTags != nil && len(r.config.CommonTags) > 0 {
		newId = newId.WithTags(r.config.CommonTags)
	}

	return newId
}

// Counter calls NewId() with the name and tags, and then calls r.CounterWithId()
// using that *Id.
func (r *spectatordRegistry) Counter(name string, tags map[string]string) *meter.Counter {
	return meter.NewCounter(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) CounterWithId(id *meter.Id) *meter.Counter {
	return meter.NewCounter(id, r.writer)
}

// MonotonicCounter calls NewId() with the name and tags, and then calls r.MonotonicCounterWithId()
// using that *Id.
func (r *spectatordRegistry) MonotonicCounter(name string, tags map[string]string) *meter.MonotonicCounter {
	return meter.NewMonotonicCounter(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) MonotonicCounterWithId(id *meter.Id) *meter.MonotonicCounter {
	return meter.NewMonotonicCounter(id, r.writer)
}

// Timer calls NewId() with the name and tags, and then calls r.TimerWithId() using that *Id.
func (r *spectatordRegistry) Timer(name string, tags map[string]string) *meter.Timer {
	return meter.NewTimer(r.NewId(name, tags), r.writer)
}

// TimerWithId returns a new *Timer, using the provided meter identifier.
func (r *spectatordRegistry) TimerWithId(id *meter.Id) *meter.Timer {
	return meter.NewTimer(id, r.writer)
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

func (r *spectatordRegistry) AgeGauge(name string, tags map[string]string) *meter.AgeGauge {
	return meter.NewAgeGauge(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) AgeGaugeWithId(id *meter.Id) *meter.AgeGauge {
	return meter.NewAgeGauge(id, r.writer)
}

func (r *spectatordRegistry) MaxGauge(name string, tags map[string]string) *meter.MaxGauge {
	return meter.NewMaxGauge(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) MaxGaugeWithId(id *meter.Id) *meter.MaxGauge {
	return meter.NewMaxGauge(id, r.writer)
}

func (r *spectatordRegistry) DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary {
	return meter.NewDistributionSummary(r.NewId(name, tags), r.writer)
}

func (r *spectatordRegistry) DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary {
	return meter.NewDistributionSummary(id, r.writer)
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

func (r *spectatordRegistry) GetWriter() writer.Writer {
	return r.writer
}

func (r *spectatordRegistry) Close() {
	err := r.writer.Close()

	if err != nil {
		r.GetLogger().Errorf("Error closing writer: %v", err)
	}
}
