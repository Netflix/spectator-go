// Package spectator provides a minimal Go implementation of the Netflix Java
// Spectator library. The goal of this package is to allow Go programs to emit
// metrics to Atlas.
//
// Please refer to the Java Spectator documentation for information on
// spectator / Atlas fundamentals: https://netflix.github.io/spectator/en/latest/
package spectator

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/sync/semaphore"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Meter represents the functionality presented by the individual meter types.
type Meter interface {
	MeterId() *Id
	Measure() []Measurement
}

// Config represents the Registry's configuration.
type Config struct {
	Frequency      time.Duration     `json:"frequency"`
	Timeout        time.Duration     `json:"timeout"`
	Uri            string            `json:"uri"`
	BatchSize      int               `json:"batch_size"`
	CommonTags     map[string]string `json:"common_tags"`
	PublishWorkers int64             `json:"publish_workers"`
	Log            Logger
	IsEnabled      func() bool
	IpcTimerRecord func(registry *Registry, id *Id, duration time.Duration)
}

// Registry is the collection of meters being reported.
type Registry struct {
	clock          Clock
	config         *Config
	meters         map[string]Meter
	started        bool
	debugPayload   bool
	mutex          *sync.Mutex
	http           *HttpClient
	sentMetrics    *Counter
	invalidMetrics *Counter
	droppedHttp    *Counter
	droppedOther   *Counter
	quit           chan struct{}
	publishSync    *semaphore.Weighted
}

// NewRegistryConfiguredBy loads a new Config JSON file from disk at the path
// specified.
//
// Please note, when this method is used to load the configuration both
// Config.Frequency and Config.Timeout are assumed to not be a time.Duration but
// an int64 with second precision. As such this function multiplies those
// configuration values by time.Second, to convert them to time.Duration values.
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

	config.Timeout *= time.Second
	config.Frequency *= time.Second
	return NewRegistry(&config), nil
}

// NewRegistry generates a new registry from the config.
//
// If the config.IpcTimerRecord is unset, a default implementation is used.
//
// If config.IsEnabled is unset, it defaults to an implementation that returns
// true.
//
// If config.Log is unset, it defaults to using the default logger.
func NewRegistry(config *Config) *Registry {
	if config.IpcTimerRecord == nil {
		config.IpcTimerRecord = func(registry *Registry, id *Id, duration time.Duration) {
			registry.TimerWithId(id).Record(duration)
		}
	}
	if config.IsEnabled == nil {
		config.IsEnabled = func() bool { return true }
	}
	if config.Log == nil {
		config.Log = defaultLogger()
	}

	if config.PublishWorkers < 1 {
		config.PublishWorkers = 1
	}

	r := &Registry{
		&SystemClock{}, config,
		map[string]Meter{},
		false,
		debugPayload(),
		&sync.Mutex{}, nil, nil, nil, nil, nil,
		make(chan struct{}),
		semaphore.NewWeighted(config.PublishWorkers),
	}
	r.http = NewHttpClient(r, r.config.Timeout)
	r.sentMetrics = r.Counter("spectator.measurements",
		map[string]string{"id": "sent"})
	r.invalidMetrics = r.Counter("spectator.measurements",
		map[string]string{"id": "dropped", "error": "validation"})
	r.droppedHttp = r.Counter("spectator.measurements",
		map[string]string{"id": "dropped", "error": "http-error"})
	r.droppedOther = r.Counter("spectator.measurements",
		map[string]string{"id": "dropped", "error": "other"})

	return r
}

// NewRegistryWithClock returns a new registry with the clock overriden to the
// one injected here. This function is mostly used for testing.
func NewRegistryWithClock(config *Config, clock Clock) *Registry {
	r := NewRegistry(config)
	r.clock = clock
	return r
}

// GetLogger returns the internal logger.
func (r *Registry) GetLogger() Logger {
	return r.config.Log
}

// Meters returns all the internal meters.
func (r *Registry) Meters() []Meter {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	meters := make([]Meter, 0, len(r.meters))
	for _, m := range r.meters {
		meters = append(meters, m)
	}
	return meters
}

// Clock returns the internal clock.
func (r *Registry) Clock() Clock {
	return r.clock
}

// SetLogger overrides the internal logger.
func (r *Registry) SetLogger(logger Logger) {
	r.config.Log = logger
}

// Start spins-up the background goroutine(s) for emitting collected metrics.
func (r *Registry) Start() error {
	if r.config == nil || r.config.Uri == "" {
		err := "registry config has no uri. Ignoring Start request"
		r.config.Log.Infof(err)
		return fmt.Errorf(err)
	}
	if r.started {
		err := "registry has already started. Ignoring Start request"
		r.config.Log.Infof(err)
		return fmt.Errorf(err)
	}

	r.started = true
	r.quit = make(chan struct{})
	ticker := time.NewTicker(r.config.Frequency)
	go func() {
		for {
			select {
			case <-ticker.C:
				// send measurements
				r.config.Log.Debugf("Sending measurements")
				r.publish()
			case <-r.quit:
				ticker.Stop()
				r.config.Log.Infof("Send last updates and quit")
				return
			}
		}
	}()

	return nil
}

// Stop shuts down the running goroutine(s), and attempts to flush the metrics.
func (r *Registry) Stop() {
	close(r.quit)
	r.started = false
	// flush metrics
	r.publish()
}

func debugPayload() bool {
	if value, ok := os.LookupEnv("SPECTATOR_DEBUG_PAYLOAD"); ok {
		if value == "1" {
			return true
		}
	}
	return false
}

func shouldSendMeasurement(measurement Measurement) bool {
	v := measurement.value
	if math.IsNaN(v) {
		return false
	}
	isGauge := opFromTags(measurement.id.tags) == maxOp
	return isGauge || v >= 0
}

// Measurements returns the list of internal measurements that should be sent.
func (r *Registry) Measurements() []Measurement {
	var measurements []Measurement
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, meter := range r.meters {
		for _, measure := range meter.Measure() {
			if shouldSendMeasurement(measure) {
				measurements = append(measurements, measure)
			}
		}
	}
	return measurements
}

type atlasMessage struct {
	Type       string   `json:"type"`
	ErrorCount int      `json:"errorCount"`
	Message    []string `json:"message"`
}

func getUnique(messages []string) string {
	uniq := map[string]struct{}{}
	for _, msg := range messages {
		uniq[msg] = struct{}{}
	}
	keys := make([]string, 0, len(uniq))
	for key := range uniq {
		keys = append(keys, key)
	}
	return strings.Join(keys, "; ")
}

func (r *Registry) sendBatch(measurements []Measurement) {
	numMeasurements := int64(len(measurements))
	r.config.Log.Debugf("Sending %d measurements to %s", len(measurements), r.config.Uri)
	if r.debugPayload {
		for _, m := range measurements {
			r.config.Log.Debugf("reporting: %s", m.String())
		}
	}
	jsonBytes, err := r.measurementsToJson(measurements)
	if err != nil {
		r.droppedOther.Add(numMeasurements)
		r.config.Log.Errorf("Unable to convert measurements to json: %v", err)
	} else {
		var resp HttpResponse
		resp, err = r.http.PostJson(r.config.Uri, jsonBytes)
		if err != nil {
			r.droppedHttp.Add(numMeasurements)
		} else {
			sent := int64(0)
			dropped := int64(0)
			if resp.Status == 200 {
				sent = numMeasurements
			} else if resp.Status < 500 {
				var atlasResponse atlasMessage
				err = json.Unmarshal(resp.Body, &atlasResponse)
				if err != nil {
					r.config.Log.Errorf("%d measurement(s) dropped. Http status: %d", numMeasurements, resp.Status)
					r.droppedOther.Add(numMeasurements)
				} else {
					dropped = int64(atlasResponse.ErrorCount)
					sent = numMeasurements - dropped
					// get the unique error messages
					var errorMsg string
					if len(atlasResponse.Message) > 0 {
						errorMsg = getUnique(atlasResponse.Message)
					} else {
						errorMsg = "unknown cause"
					}
					r.config.Log.Infof("%d measurement(s) dropped due to validation errors: %s",
						dropped, errorMsg)
				}
			} else {
				// 5xx error from server, note that sent and dropped are 0
				r.droppedHttp.Add(numMeasurements)
			}
			r.invalidMetrics.Add(dropped)
			r.sentMetrics.Add(sent)
		}
	}
}

func (r *Registry) publish() {
	if len(r.config.Uri) == 0 {
		return
	}

	measurements := r.Measurements()
	r.config.Log.Debugf("Got %d measurements", len(measurements))
	if !r.config.IsEnabled() {
		return
	}

	wg := new(sync.WaitGroup)
	for i := 0; i < len(measurements); i += r.config.BatchSize {
		end := i + r.config.BatchSize
		if end > len(measurements) {
			end = len(measurements)
		}
		err := r.publishSync.Acquire(context.Background(), 1)
		if err != nil {
			r.config.Log.Errorf("Unable to acquire semaphore: %v.", err)
			return
		}
		wg.Add(1)
		go func(m []Measurement) {
			r.sendBatch(m)
			r.publishSync.Release(1)
			wg.Done()
		}(measurements[i:end])
	}

	wg.Wait()
}

func (r *Registry) buildStringTable(payload *[]interface{}, measurements []Measurement) map[string]int {
	strTable := make(map[string]int)
	commonTags := r.config.CommonTags
	for k, v := range commonTags {
		strTable[k] = 0
		strTable[v] = 0
	}

	strTable["name"] = 0
	for _, measure := range measurements {
		strTable[measure.id.name] = 0
		for k, v := range measure.id.tags {
			strTable[k] = 0
			strTable[v] = 0
		}
	}
	sortedStrings := make([]string, 0, len(strTable))
	for s := range strTable {
		sortedStrings = append(sortedStrings, s)
	}
	sort.Strings(sortedStrings)
	for i, s := range sortedStrings {
		strTable[s] = i
	}
	*payload = append(*payload, len(strTable))
	// can't append the strings in one call since we can't convert []string to []interface{}
	for _, s := range sortedStrings {
		*payload = append(*payload, s)
	}

	return strTable
}

const (
	addOp = 0
	maxOp = 10
)

func opFromTags(tags map[string]string) int {
	switch tags["statistic"] {
	case "count", "totalAmount", "totalTime", "totalOfSquares", "percentile":
		return addOp
	default:
		return maxOp
	}
}

func (r *Registry) appendMeasurement(payload *[]interface{}, strings map[string]int, m Measurement) {
	op := opFromTags(m.id.tags)
	commonTags := r.config.CommonTags
	*payload = append(*payload, len(m.id.tags)+1+len(commonTags))
	for k, v := range commonTags {
		*payload = append(*payload, strings[k])
		*payload = append(*payload, strings[v])
	}
	for k, v := range m.id.tags {
		*payload = append(*payload, strings[k])
		*payload = append(*payload, strings[v])
	}
	*payload = append(*payload, strings["name"])
	*payload = append(*payload, strings[m.id.name])
	*payload = append(*payload, op)
	*payload = append(*payload, m.value)
}

func (r *Registry) measurementsToJson(measurements []Measurement) ([]byte, error) {
	var payload []interface{}
	stringTable := r.buildStringTable(&payload, measurements)
	for _, m := range measurements {
		r.appendMeasurement(&payload, stringTable, m)
	}

	return json.Marshal(payload)
}

// MeterFactoryFun is a type to allow dependency injection of the function used to generate meters.
type MeterFactoryFun func() Meter

// NewMeter registers a new meter internally, and then returns it to the caller.
// The meterFactory is used to generate this meter. If the id.MapKey() is
// already present in the internal collection, that is returned instead of
// creating a new one.
func (r *Registry) NewMeter(id *Id, meterFactory MeterFactoryFun) Meter {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	meter, exists := r.meters[id.MapKey()]
	if !exists {
		meter = meterFactory()
		r.meters[id.MapKey()] = meter
	}
	return meter
}

// NewId calls spectator.NewId().
func (r *Registry) NewId(name string, tags map[string]string) *Id {
	return NewId(name, tags)
}

// CounterWithId returns a new *Counter, using the provided meter identifier.
func (r *Registry) CounterWithId(id *Id) *Counter {
	m := r.NewMeter(id, func() Meter {
		return NewCounter(id)
	})

	c, ok := m.(*Counter)
	if ok {
		return c
	}

	r.config.Log.Errorf("Unable to register a counter with id=%v - a meter %v exists", id, c)

	// should throw in strict mode
	return NewCounter(id)
}

// Counter calls NewId() with the name and tags, and then calls r.CounterWithId()
// using that *Id.
func (r *Registry) Counter(name string, tags map[string]string) *Counter {
	return r.CounterWithId(NewId(name, tags))
}

// TimerWithId returns a new *Timer, using the provided meter identifier.
func (r *Registry) TimerWithId(id *Id) *Timer {
	m := r.NewMeter(id, func() Meter {
		return NewTimer(id)
	})

	t, ok := m.(*Timer)
	if ok {
		return t
	}

	r.config.Log.Errorf("Unable to register a timer with %v - a meter %v exists", id, t)

	// throw in strict mode
	return NewTimer(id)
}

// Timer calls NewId() with the name and tags, and then calls r.TimerWithId()
// using that *Id.
func (r *Registry) Timer(name string, tags map[string]string) *Timer {
	return r.TimerWithId(NewId(name, tags))
}

// GaugeWithId returns a new *Gauge, using the provided meter identifier.
func (r *Registry) GaugeWithId(id *Id) *Gauge {
	m := r.NewMeter(id, func() Meter {
		return NewGauge(id)
	})

	g, ok := m.(*Gauge)
	if ok {
		return g
	}

	r.config.Log.Errorf("Unable to register a gauge with id=%v - a meter %v exists", id, g)

	// throw in strict mode
	return NewGauge(id)
}

// Gauge calls NewId() with the name and tags, and then calls r.GaugeWithId()
// using that *Id.
func (r *Registry) Gauge(name string, tags map[string]string) *Gauge {
	return r.GaugeWithId(NewId(name, tags))
}

// DistributionSummaryWithId returns a new *DistributionSummary, using the
// provided meter identifier.
func (r *Registry) DistributionSummaryWithId(id *Id) *DistributionSummary {
	m := r.NewMeter(id, func() Meter {
		return NewDistributionSummary(id)
	})

	d, ok := m.(*DistributionSummary)
	if ok {
		return d
	}

	r.config.Log.Errorf("Unable to register a distribution summary with id=%v - a meter %v exists", id, d)

	// throw in strict mode
	return NewDistributionSummary(id)
}

// DistributionSummary calls NewId() using the name and tags, and then calls
// r.DistributionSummaryWithId() using that *Id.
func (r *Registry) DistributionSummary(name string, tags map[string]string) *DistributionSummary {
	return r.DistributionSummaryWithId(NewId(name, tags))
}
