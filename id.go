package spectator

import (
	"bytes"
	"fmt"
	"sort"
)

// Id represents a meter's identifying information and dimensions (tags).
type Id struct {
	name string
	tags map[string]string
	key  string
}

// MapKey computes and saves a key within the struct to be used to uniquely
// identify this *Id in a map. This does use the information from within the
// *Id, so it assumes you've not accidentally double-declared this *Id.
func (id *Id) MapKey() string {
	if len(id.key) > 0 {
		return id.key
	}

	var buf bytes.Buffer
	_, err := buf.WriteString(id.name)
	const errKey = "ERR"
	if err != nil {
		return errKey
	}
	var keys []string
	for k := range id.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := id.tags[k]
		_, err = buf.WriteRune('|')
		if err != nil {
			return errKey
		}
		_, err = buf.WriteString(k)
		if err != nil {
			return errKey
		}
		_, err = buf.WriteRune('|')
		if err != nil {
			return errKey
		}
		_, err = buf.WriteString(v)
		if err != nil {
			return errKey
		}
	}
	id.key = buf.String()
	return id.key
}

// NewId generates a new *Id from the metric name, and the tags you want to
// include on your metric.
func NewId(name string, tags map[string]string) *Id {
	myTags := make(map[string]string)
	for k, v := range tags {
		myTags[k] = v
	}
	return &Id{name, myTags, ""}
}

// WithTag creates a deep copy of the *Id, adding the requested tag to the
// internal collection.
func (id *Id) WithTag(key string, value string) *Id {
	newTags := make(map[string]string)

	for k, v := range id.tags {
		newTags[k] = v
	}
	newTags[key] = value

	return NewId(id.name, newTags)
}

// WithStat is id.WithTag("statistic", stat). See that method's documentation
// for more info.
func (id *Id) WithStat(stat string) *Id {
	return id.WithTag("statistic", stat)
}

// WithDefaultStat is effectively the WithStat() method, except it only creates
// the deep copy if the "statistic" tag is not set or is set to empty string. If
// the "statistic" tag is already present, the *Id is returned without being
// copied.
func (id *Id) WithDefaultStat(stat string) *Id {
	s := id.tags["statistic"]
	if s == "" {
		return id.WithTag("statistic", stat)
	} else {
		return id
	}
}

func (id *Id) String() string {
	return fmt.Sprintf("Id{name=%s,tags=%v}", id.name, id.tags)
}

// Name exposes the internal metric name field.
func (id *Id) Name() string {
	return id.name
}

// Tags directly exposes the internal tags map. This is not a copy of the map,
// so any modifications to it will be observed by the *Id.
func (id *Id) Tags() map[string]string {
	return id.tags
}

// WithTags takes a map of tags, and returns a deep copy of *Id with the new
// tags appended to the original ones. Overlapping keys are overwritten. If the
// input to this method is empty, this does not return a deep copy of *Id.
func (id *Id) WithTags(tags map[string]string) *Id {
	if len(tags) == 0 {
		return id
	}

	newTags := make(map[string]string)

	for k, v := range id.tags {
		newTags[k] = v
	}

	for k, v := range tags {
		newTags[k] = v
	}
	return NewId(id.name, newTags)
}
