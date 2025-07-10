package meter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Id represents a meter's identifying information and dimensions (tags).
type Id struct {
	name string
	tags map[string]string
	// keyOnce protects access to key, allowing it to be computed on demand
	// without racing other readers.
	keyOnce sync.Once
	key     string
	// spectatordId is the Id formatted for spectatord line protocol
	spectatordId string
}

var builderPool = &sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

// MapKey computes and saves a key within the struct to be used to uniquely
// identify this *Id in a map. This does use the information from within the
// *Id, so it assumes you've not accidentally double-declared this *Id.
func (id *Id) MapKey() string {
	id.keyOnce.Do(func() {
		// if the key was set directly during Id construction, then do not
		// compute a value.
		if id.key != "" {
			return
		}

		buf := builderPool.Get().(*strings.Builder)
		buf.Reset()
		defer builderPool.Put(buf)

		const errKey = "ERR"
		id.key = func() string {
			_, err := buf.WriteString(id.name)
			if err != nil {
				return errKey
			}
			keys := make([]string, 0, len(id.tags))
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
			return buf.String()
		}()
	})
	return id.key
}

// NewId generates a new *Id from the metric name, and the tags you want to
// include on your metric.
func NewId(name string, tags map[string]string) *Id {
	myTags := make(map[string]string)
	for k, v := range tags {
		myTags[k] = v
	}

	spectatorId := toSpectatorId(name, tags)

	return &Id{
		name:         name,
		tags:         myTags,
		spectatordId: spectatorId,
	}
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

func toSpectatorId(name string, tags map[string]string) string {
	var sb strings.Builder
	writeSanitized(&sb, name)

	// Append sanitized keys and values.
	for k, v := range tags {
		sb.WriteString(",")
		writeSanitized(&sb, k)
		sb.WriteString("=")
		writeSanitized(&sb, v)
	}

	return sb.String()
}

func writeSanitized(sb *strings.Builder, input string) {
	for _, r := range input {
		if !isValidCharacter(r) {
			sb.WriteRune('_')
		} else {
			sb.WriteRune(r)
		}
	}
}

func isValidCharacter(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' ||
		r == '.' ||
		r == '_' ||
		r == '~' ||
		r == '^'
}
