package meter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type tagPair struct {
	key   string
	value string
}

type tagPairs []tagPair

func newTagPairs(tags map[string]string) tagPairs {
	if len(tags) == 0 {
		return nil
	}

	flat := make(tagPairs, 0, len(tags))
	for k, v := range tags {
		flat = append(flat, tagPair{key: k, value: v})
	}
	return flat
}

func (tags tagPairs) clone(extraPairs int) tagPairs {
	cloned := make(tagPairs, len(tags), len(tags)+extraPairs)
	copy(cloned, tags)
	return cloned
}

func (tags tagPairs) upsert(key string, value string) tagPairs {
	for i := range tags {
		if tags[i].key == key {
			tags[i].value = value
			return tags
		}
	}
	return append(tags, tagPair{key: key, value: value})
}

func (tags tagPairs) sorted() []tagPair {
	pairs := make([]tagPair, len(tags))
	copy(pairs, tags)
	sort.Slice(pairs, func(i int, j int) bool {
		return pairs[i].key < pairs[j].key
	})
	return pairs
}

func (tags tagPairs) toMap() map[string]string {
	m := make(map[string]string, len(tags))
	for i := range tags {
		m[tags[i].key] = tags[i].value
	}
	return m
}

// Id represents a meter's identifying information and dimensions (tags).
type Id struct {
	name string
	// tags stores key/value pairs in a slice to avoid the allocation overhead
	// of a map while keeping the representation explicit.
	tags tagPairs
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

			pairs := id.tags.sorted()
			if len(pairs) == 0 {
				return buf.String()
			}

			for i := range pairs {
				_, err = buf.WriteRune('|')
				if err != nil {
					return errKey
				}
				_, err = buf.WriteString(pairs[i].key)
				if err != nil {
					return errKey
				}
				_, err = buf.WriteRune('|')
				if err != nil {
					return errKey
				}
				_, err = buf.WriteString(pairs[i].value)
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
	pairs := newTagPairs(tags)

	return &Id{
		name:         name,
		tags:         pairs,
		spectatordId: toSpectatorIdFromPairs(name, pairs),
	}
}

// newIdFromPairs creates an *Id directly from a pre-built tag slice,
// avoiding an intermediate map allocation. Used by WithTag and WithTags.
func newIdFromPairs(name string, tags tagPairs) *Id {
	return &Id{
		name:         name,
		tags:         tags,
		spectatordId: toSpectatorIdFromPairs(name, tags),
	}
}

// WithTag creates a deep copy of the *Id, adding the requested tag to the
// internal collection.
func (id *Id) WithTag(key string, value string) *Id {
	newTags := id.tags.clone(1).upsert(key, value)
	return newIdFromPairs(id.name, newTags)
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
	return id.tags.toMap()
}

// WithTags takes a map of tags, and returns a deep copy of *Id with the new
// tags appended to the original ones. Overlapping keys are overwritten. If the
// input to this method is empty, this does not return a deep copy of *Id.
func (id *Id) WithTags(tags map[string]string) *Id {
	if len(tags) == 0 {
		return id
	}

	newTags := id.tags.clone(len(tags))
	for k, v := range tags {
		newTags = newTags.upsert(k, v)
	}
	return newIdFromPairs(id.name, newTags)
}

// toSpectatorIdFromPairs builds the spectatord line-protocol name from tag
// pairs. Reuses a pooled buffer to avoid builder
// growth allocations; the only allocation is the returned string.
func toSpectatorIdFromPairs(name string, tags tagPairs) string {
	bp := byteBufPool.Get().(*[]byte)
	b := (*bp)[:0]

	b = appendSanitized(b, name)

	for i := range tags {
		b = append(b, ',')
		b = appendSanitized(b, tags[i].key)
		b = append(b, '=')
		b = appendSanitized(b, tags[i].value)
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
