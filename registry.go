package spectator

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"sort"
	"sync"
	"time"
)

type Meter interface {
	MeterId() *Id
	Measure() []Measurement
}

type Config struct {
	Frequency  time.Duration     `json:"frequency"`
	Timeout    time.Duration     `json:"timeout"`
	Uri        string            `json:"uri"`
	CommonTags map[string]string `json:"common_tags"`
}

type Registry struct {
	clock   Clock
	config  *Config
	meters  map[string]Meter
	started bool
	log     Logger
	mutex   *sync.Mutex
	http    *HttpClient
	quit    chan struct{}
}

func NewRegistryConfiguredBy(filePath string) (*Registry, error) {
	bytes, err := ioutil.ReadFile(filePath)
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

func NewRegistry(config *Config) *Registry {
	r := &Registry{&SystemClock{}, config, map[string]Meter{}, false,
		defaultLogger(), &sync.Mutex{}, nil, make(chan struct{})}
	r.http = NewHttpClient(r, r.config.Timeout)
	return r
}

func (r *Registry) Meters() []Meter {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	meters := make([]Meter, 0, len(r.meters))
	for _, m := range r.meters {
		meters = append(meters, m)
	}
	return meters
}

func (r *Registry) Clock() Clock {
	return r.clock
}

func (r *Registry) SetLogger(logger Logger) {
	r.log = logger
}

func (r *Registry) Start() {
	if r.config == nil || r.config.Uri == "" {
		r.log.Infof("Registry config has no uri.  Ingnoring Start request.")
		return
	}
	if r.started {
		r.log.Infof("Registry has already started. Ignoring Start request.")
		return
	}

	r.started = true
	r.quit = make(chan struct{})
	ticker := time.NewTicker(time.Duration(r.config.Frequency))
	go func() {
		for {
			select {
			case <-ticker.C:
				// send measurements
				r.log.Infof("Sending measurements")
				r.publish()
			case <-r.quit:
				ticker.Stop()
				r.log.Infof("Send last updates and quit")
				return
			}
		}
	}()
}

func (r *Registry) Stop() {
	close(r.quit)
	r.started = false
	// flush metrics
	r.publish()
}

func shouldSendMeasurement(measurement Measurement) bool {
	v := measurement.value
	if math.IsNaN(v) {
		return false
	}
	s := measurement.id.tags["statistic"]
	return s == "gauge" || v > 0
}

func (r *Registry) publish() {
	if len(r.config.Uri) == 0 {
		return
	}

	var measurements []Measurement

	r.mutex.Lock()
	{
		for _, meter := range r.meters {
			for _, measure := range meter.Measure() {
				if shouldSendMeasurement(measure) {
					measurements = append(measurements, measure)
				}
			}
		}
		r.mutex.Unlock()
	}
	r.log.Debugf("Measurements: %v", measurements)
	jsonBytes, err := r.measurementsToJson(measurements)
	if err != nil {
		r.log.Errorf("Unable to convert measurements to json: %v", err)
	} else {
		r.http.PostJson(r.config.Uri, jsonBytes)
	}
}

func (r *Registry) buildStringTable(payload *[]interface{}, measurements []Measurement) map[string]int {
	var strings = make(map[string]int)
	commonTags := r.config.CommonTags
	for k, v := range commonTags {
		strings[k] = 0
		strings[v] = 0
	}

	strings["name"] = 0
	for _, measure := range measurements {
		strings[measure.id.name] = 0
		for k, v := range measure.id.tags {
			strings[k] = 0
			strings[v] = 0
		}
	}
	sortedStrings := make([]string, 0, len(strings))
	for s := range strings {
		sortedStrings = append(sortedStrings, s)
	}
	sort.Strings(sortedStrings)
	for i, s := range sortedStrings {
		strings[s] = i
	}
	*payload = append(*payload, len(strings))
	// can't append the strings in one call since we can't convert []string to []interface{}
	for _, s := range sortedStrings {
		*payload = append(*payload, s)
	}

	return strings
}

const (
	unknownOp = -1
	addOp     = 0
	maxOp     = 10
)

func opFromTags(tags map[string]string) int {
	switch tags["statistic"] {
	case "count", "totalAmount", "totalTime", "totalOfSquares", "percentile":
		return addOp
	case "max", "gauge", "activeTasks", "duration":
		return maxOp
	default:
		return unknownOp
	}
}

func (r *Registry) appendMeasurement(payload *[]interface{}, strings map[string]int, m Measurement) {
	op := opFromTags(m.id.tags)
	if op != unknownOp {
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
	} else {
		r.log.Infof("Invalid statistic for id=%v", m.id)
	}

}

func (r *Registry) measurementsToJson(measurements []Measurement) ([]byte, error) {
	var payload []interface{}
	strings := r.buildStringTable(&payload, measurements)
	for _, m := range measurements {
		r.appendMeasurement(&payload, strings, m)
	}

	return json.Marshal(payload)
}

type meterFactoryFun func() Meter

func (r *Registry) newMeter(id *Id, meterFactory meterFactoryFun) Meter {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	meter, exists := r.meters[id.mapKey()]
	if !exists {
		meter = meterFactory()
		r.meters[id.mapKey()] = meter
	}
	return meter
}

func (r *Registry) NewId(name string, tags map[string]string) *Id {
	return newId(name, tags)
}

func (r *Registry) CounterWithId(id *Id) *Counter {
	m := r.newMeter(id, func() Meter {
		return NewCounter(id)
	})

	c, ok := m.(*Counter)
	if ok {
		return c
	}

	r.log.Errorf("Unable to register a counter with id=%v - a meter %v exists", id, c)

	// should throw in strict mode
	return NewCounter(id)
}

func (r *Registry) Counter(name string, tags map[string]string) *Counter {
	return r.CounterWithId(newId(name, tags))
}

func (r *Registry) TimerWithId(id *Id) *Timer {
	m := r.newMeter(id, func() Meter {
		return NewTimer(id)
	})

	t, ok := m.(*Timer)
	if ok {
		return t
	}

	r.log.Errorf("Unable to register a timer with name=%s,tags=%v - a meter %v exists", id, t)

	// throw in strict mode
	return NewTimer(id)
}

func (r *Registry) Timer(name string, tags map[string]string) *Timer {
	return r.TimerWithId(newId(name, tags))
}

func (r *Registry) GaugeWithId(id *Id) *Gauge {
	m := r.newMeter(id, func() Meter {
		return NewGauge(id)
	})

	g, ok := m.(*Gauge)
	if ok {
		return g
	}

	r.log.Errorf("Unable to register a gauge with id=%v - a meter %v exists", id, g)

	// throw in strict mode
	return NewGauge(id)
}

func (r *Registry) Gauge(name string, tags map[string]string) *Gauge {
	return r.GaugeWithId(newId(name, tags))
}

func (r *Registry) DistributionSummaryWithId(id *Id) *DistributionSummary {
	m := r.newMeter(id, func() Meter {
		return NewDistributionSummary(id)
	})

	d, ok := m.(*DistributionSummary)
	if ok {
		return d
	}

	r.log.Errorf("Unable to register a distribution summary with id=%v - a meter %v exists", id, d)

	// throw in strict mode
	return NewDistributionSummary(id)
}

func (r *Registry) DistributionSummary(name string, tags map[string]string) *DistributionSummary {
	return r.DistributionSummaryWithId(newId(name, tags))
}
