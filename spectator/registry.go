// Package spectator provides a minimal Go implementation of the Netflix Java
// Spectator library. The goal of this package is to allow Go programs to emit
// metrics to Atlas.
//
// Please refer to the Java Spectator documentation for information on
// spectator / Atlas fundamentals: https://netflix.github.io/spectator/en/latest/
package spectator

import (
	"encoding/json"
	"github.com/Netflix/spectator-go/spectator/meter"
	"github.com/Netflix/spectator-go/spectator/writer"
	"io/ioutil"
	"path/filepath"
)

// Config represents the Registry's configuration.
type Config struct {
	CommonTags map[string]string `json:"common_tags"`
	Log        Logger
}

// Meter represents the functionality presented by the individual meter types.
type Meter interface {
	MeterId() *meter.Id
}

// RegistryInterface was extracted from Registry. It's used to make sure we're honoring the same API for thin and fat client, might delete later.
type RegistryInterface interface {
	GetLogger() Logger
	SetLogger(logger Logger)
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

// TODO The New* methods will remain as is. Only the Registry instantiated should be changed.

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

	return NewRegistry(&config), nil
}

// NewRegistry generates a new registry from the config.
//
// If config.IsEnabled is unset, it defaults to an implementation that returns
// true.
//
// If config.Log is unset, it defaults to using the default logger.
func NewRegistry(config *Config) *Registry {
	if config.Log == nil {
		config.Log = defaultLogger()
	}

	printWriter := &writer.PrintWriter{}
	r := &Registry{
		config: config,
		writer: printWriter,
		quit:   make(chan struct{}),
	}

	return r
}

// GetLogger returns the internal logger.
func (r *Registry) GetLogger() Logger {
	return r.config.Log
}

func (r *Registry) SetLogger(logger Logger) {
	r.config.Log = logger
}

// TODO investigate how to provide the same functionality for MeterFactory. Do we need to expose the writer?
// NewMeter creates a new Meter using the provided meterFactory
func (r *Registry) NewMeter(id *meter.Id, meterFactory MeterFactoryFun) Meter {
	return meterFactory()
}

// NewId calls spectator.NewId().
func (r *Registry) NewId(name string, tags map[string]string) *meter.Id {
	return meter.NewId(name, tags)
}

// TODO append common tags upon creation

// Counter calls NewId() with the name and tags, and then calls r.CounterWithId()
// using that *Id.
func (r *Registry) Counter(name string, tags map[string]string) *meter.Counter {
	return meter.NewCounter(meter.NewId(name, tags), r.writer)
}

func (r *Registry) CounterWithId(id *meter.Id) *meter.Counter {
	return meter.NewCounter(id, r.writer)
}

// Timer calls NewId() with the name and tags, and then calls r.TimerWithId() using that *Id.
func (r *Registry) Timer(name string, tags map[string]string) *meter.Timer {
	return meter.NewTimer(meter.NewId(name, tags), r.writer)
}

// TimerWithId returns a new *Timer, using the provided meter identifier.
func (r *Registry) TimerWithId(id *meter.Id) *meter.Timer {
	return meter.NewTimer(id, r.writer)
}

func (r *Registry) Gauge(name string, tags map[string]string) *meter.Gauge {
	return meter.NewGauge(meter.NewId(name, tags), r.writer)
}

func (r *Registry) GaugeWithId(id *meter.Id) *meter.Gauge {
	return meter.NewGauge(id, r.writer)
}

func (r *Registry) MaxGauge(name string, tags map[string]string) *meter.MaxGauge {
	return meter.NewMaxGauge(meter.NewId(name, tags), r.writer)
}

func (r *Registry) MaxGaugeWithId(id *meter.Id) *meter.MaxGauge {
	return meter.NewMaxGauge(id, r.writer)
}

func (r *Registry) DistributionSummary(name string, tags map[string]string) *meter.DistributionSummary {
	return meter.NewDistributionSummary(meter.NewId(name, tags), r.writer)
}

func (r *Registry) DistributionSummaryWithId(id *meter.Id) *meter.DistributionSummary {
	return meter.NewDistributionSummary(id, r.writer)
}

func (r *Registry) PercentileDistributionSummary(name string, tags map[string]string) *meter.PercentileDistributionSummary {
	return meter.NewPercentileDistributionSummary(meter.NewId(name, tags), r.writer)
}

func (r *Registry) PercentileDistributionSummaryWithId(id *meter.Id) *meter.PercentileDistributionSummary {
	return meter.NewPercentileDistributionSummary(id, r.writer)
}

func (r *Registry) PercentileTimer(name string, tags map[string]string) *meter.PercentileTimer {
	return meter.NewPercentileTimer(meter.NewId(name, tags), r.writer)
}

func (r *Registry) PercentileTimerWithId(id *meter.Id) *meter.PercentileTimer {
	return meter.NewPercentileTimer(id, r.writer)
}

// Stop shuts down the running goroutine(s), and attempts to flush the metrics.
func (r *Registry) Stop() {
	// TODO implement
	close(r.quit)
}

// TODO do we need this?
//func shouldSendMeasurement(measurement Measurement) bool {
//	v := measurement.value
//	if math.IsNaN(v) {
//		return false
//	}
//	isGauge := opFromTags(measurement.id.tags) == maxOp
//	return isGauge || v >= 0
//}
//
//const (
//	addOp = 0
//	maxOp = 10
//)
//
//func opFromTags(tags map[string]string) int {
//	switch tags["statistic"] {
//	case "count", "totalAmount", "totalTime", "totalOfSquares", "percentile":
//		return addOp
//	default:
//		return maxOp
//	}
//}
