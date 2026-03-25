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
	// flatTags stores tags as a flat [k1, v1, k2, v2, ...] slice.
	// This avoids the two allocations (hmap header + initial bucket) that a
	// map[string]string would require when the map escapes to the heap.
	flatTags []string
	// keyOnce protects access to key, allowing it to be computed on demand
	// without racing other readers.
	keyOnce sync.Once
	key     string
	// spectatordId is the Id formatted for spectatord line protocol
	spectatordId string
}

var builderPool = sync.Pool{
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

			n := len(id.flatTags) / 2
			if n == 0 {
				return buf.String()
			}

			// Extract keys for sorting
			keys := make([]string, 0, n)
			for i := 0; i+1 < len(id.flatTags); i += 2 {
				keys = append(keys, id.flatTags[i])
			}
			sort.Strings(keys)

			// Build a temporary map for O(1) value lookup during key emission.
			// MapKey is called at most once per Id (result is cached), so this
			// transient allocation is acceptable.
			lookup := make(map[string]string, n)
			for i := 0; i+1 < len(id.flatTags); i += 2 {
				lookup[id.flatTags[i]] = id.flatTags[i+1]
			}

			for _, k := range keys {
				v := lookup[k]
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
	var flat []string
	if len(tags) > 0 {
		flat = make([]string, 0, 2*len(tags))
		for k, v := range tags {
			flat = append(flat, k, v)
		}
	}

	return &Id{
		name:         name,
		flatTags:     flat,
		spectatordId: toSpectatorIdFromFlat(name, flat),
	}
}

// newIdFromFlat creates an *Id directly from a pre-built flat tag slice,
// avoiding an intermediate map allocation. Used by WithTag and WithTags.
func newIdFromFlat(name string, flatTags []string) *Id {
	return &Id{
		name:         name,
		flatTags:     flatTags,
		spectatordId: toSpectatorIdFromFlat(name, flatTags),
	}
}

// WithTag creates a deep copy of the *Id, adding the requested tag to the
// internal collection.
func (id *Id) WithTag(key string, value string) *Id {
	newFlat := make([]string, len(id.flatTags))
	copy(newFlat, id.flatTags)

	for i := 0; i < len(id.flatTags); i += 2 {
		if id.flatTags[i] == key {
			newFlat[i+1] = value
			return newIdFromFlat(id.name, newFlat)
		}
	}
	newFlat = append(newFlat, key, value)
	return newIdFromFlat(id.name, newFlat)
}

func (id *Id) String() string {
	return fmt.Sprintf("Id{name=%s,tags=%v}", id.name, id.Tags())
}

// Name exposes the internal metric name field.
func (id *Id) Name() string {
	return id.name
}

// Tags returns a new map containing the identifier's tags. Each call allocates
// a fresh map; mutating the returned map does not affect the *Id.
func (id *Id) Tags() map[string]string {
	m := make(map[string]string, len(id.flatTags)/2)
	for i := 0; i+1 < len(id.flatTags); i += 2 {
		m[id.flatTags[i]] = id.flatTags[i+1]
	}
	return m
}

// WithTags takes a map of tags, and returns a deep copy of *Id with the new
// tags appended to the original ones. Overlapping keys are overwritten. If the
// input to this method is empty, this does not return a deep copy of *Id.
func (id *Id) WithTags(tags map[string]string) *Id {
	if len(tags) == 0 {
		return id
	}

	newFlat := make([]string, len(id.flatTags), len(id.flatTags)+2*len(tags))
	copy(newFlat, id.flatTags)

	for k, v := range tags {
		updated := false
		for i := 0; i < len(id.flatTags); i += 2 {
			if id.flatTags[i] == k {
				newFlat[i+1] = v
				updated = true
				break
			}
		}
		if !updated {
			newFlat = append(newFlat, k, v)
		}
	}
	return newIdFromFlat(id.name, newFlat)
}

// toSpectatorIdFromFlat builds the spectatord line-protocol name from a flat
// [k1, v1, k2, v2, ...] tag slice. Reuses a pooled buffer to avoid builder
// growth allocations; the only allocation is the returned string.
func toSpectatorIdFromFlat(name string, flatTags []string) string {
	bp := byteBufPool.Get().(*[]byte)
	b := (*bp)[:0]

	b = appendSanitized(b, name)

	for i := 0; i+1 < len(flatTags); i += 2 {
		b = append(b, ',')
		b = appendSanitized(b, flatTags[i])
		b = append(b, '=')
		b = appendSanitized(b, flatTags[i+1])
	}

	result := string(b)
	*bp = b
	byteBufPool.Put(bp)
	return result
}

// appendSanitized appends the sanitized form of input to dst and returns the
// extended slice. Every rune that passes isValidCharacter (all ASCII single-byte)
// is appended directly; invalid runes become '_'.
func appendSanitized(dst []byte, input string) []byte {
	for _, r := range input {
		if isValidCharacter(r) {
			dst = append(dst, byte(r))
		} else {
			dst = append(dst, '_')
		}
	}
	return dst
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
