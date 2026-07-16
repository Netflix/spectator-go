package meter

import (
	"fmt"
	"maps"
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
	return newId(name, maps.Clone(tags))
}

// newId creates a new *Id taking ownership of the provided tags map (no copy).
func newId(name string, tags map[string]string) *Id {
	spectatorId := toSpectatorId(name, tags)

	return &Id{
		name:         name,
		tags:         tags,
		spectatordId: spectatorId,
	}
}

// WithTag creates a deep copy of the *Id, adding the requested tag to the
// internal collection.
func (id *Id) WithTag(key string, value string) *Id {
	newTags := maps.Clone(id.tags)
	if newTags == nil {
		newTags = make(map[string]string, 1)
	}
	newTags[key] = value

	return newId(id.name, newTags)
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

	newTags := maps.Clone(id.tags)
	if newTags == nil {
		newTags = make(map[string]string, len(tags))
	}
	maps.Copy(newTags, tags)
	return newId(id.name, newTags)
}

// writeSpectatorId writes the sanitized meter id ("name,key=value,...") to sb.
// tags and commonTags are merged with commonTags taking precedence on key
// collisions, matching Registry.NewId followed by Id.WithTags(extraCommonTags).
// commonTags may be nil. This is the single serializer shared by toSpectatorId
// (WithId path) and buildLinePrefix (direct path), so both entry points always
// produce the same wire format.
func writeSpectatorId(sb *strings.Builder, name string, tags, commonTags map[string]string) {
	writeSanitized(sb, name)

	// Caller tags, except any key overridden by a common tag.
	for k, v := range tags {
		if _, overridden := commonTags[k]; overridden {
			continue
		}
		sb.WriteByte(',')
		writeSanitized(sb, k)
		sb.WriteByte('=')
		writeSanitized(sb, v)
	}

	// Common tags (win on key collision).
	for k, v := range commonTags {
		sb.WriteByte(',')
		writeSanitized(sb, k)
		sb.WriteByte('=')
		writeSanitized(sb, v)
	}
}

func toSpectatorId(name string, tags map[string]string) string {
	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	defer builderPool.Put(sb)

	// Pre-size: name + per-tag overhead (comma + key + equals + value).
	sb.Grow(len(name) + len(tags)*40)
	writeSpectatorId(sb, name, tags, nil)
	return sb.String()
}

// buildLinePrefix assembles a meter's spectatord line prefix,
// "<symbol>:<sanitized id>:", in a single builder pass. It lets a meter be
// constructed from a name and tags without allocating an *Id, copying the tags
// map, or materializing a separate spectatordId string: the only allocation is
// the returned prefix. commonTags carries the registry's extraCommonTags and is
// merged in (commonTags win on collision), so meters built this way include the
// same infrastructure tags as the WithId path. The symbol is the spectatord
// meter type (e.g. "c", "g", or "g,120" for a gauge with a TTL).
func buildLinePrefix(symbol, name string, tags, commonTags map[string]string) string {
	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	defer builderPool.Put(sb)

	// symbol + ':' + name + per-tag (",key=value") + trailing ':'
	sb.Grow(len(symbol) + 2 + len(name) + (len(tags)+len(commonTags))*40)
	sb.WriteString(symbol)
	sb.WriteByte(':')
	writeSpectatorId(sb, name, tags, commonTags)
	sb.WriteByte(':')
	return sb.String()
}

// resolveMeterId returns id when the meter was built with one (the WithId path),
// otherwise reconstructs an *Id from linePrefix (the direct path). Shared by
// every meter's MeterId().
func resolveMeterId(id *Id, linePrefix string) *Id {
	if id != nil {
		return id
	}
	return idFromLinePrefix(linePrefix)
}

// idFromLinePrefix reconstructs an *Id from a meter line prefix of the form
// "<symbol>:<spectatordId>:". Meters built directly from a name and tags do not
// retain an Id, so this rebuilds one on demand for the rarely-used MeterId().
// The spectatordId is the substring between the first and last ':' — meter type
// symbols and sanitized names/tags never contain ':'. The returned tags are the
// sanitized "key=value" pairs; original pre-sanitization tag text is not
// recoverable from the prefix.
func idFromLinePrefix(linePrefix string) *Id {
	first := strings.IndexByte(linePrefix, ':')
	last := strings.LastIndexByte(linePrefix, ':')
	if first < 0 || last <= first {
		return &Id{name: linePrefix}
	}
	spectatordId := linePrefix[first+1 : last]

	name := spectatordId
	var tags map[string]string
	if comma := strings.IndexByte(spectatordId, ','); comma >= 0 {
		name = spectatordId[:comma]
		tags = make(map[string]string)
		for _, pair := range strings.Split(spectatordId[comma+1:], ",") {
			if eq := strings.IndexByte(pair, '='); eq >= 0 {
				tags[pair[:eq]] = pair[eq+1:]
			}
		}
	}
	return &Id{name: name, tags: tags, spectatordId: spectatordId}
}

func writeSanitized(sb *strings.Builder, input string) {
	// Fast path: every valid character is single-byte ASCII, so if no byte is
	// invalid the input contains no multi-byte runes and needs no rewriting.
	// Write it in one shot, avoiding a per-rune UTF-8 decode + re-encode. This
	// is the common case, since metric names and tags are usually already clean.
	if isCleanForProtocol(input) {
		sb.WriteString(input)
		return
	}

	// Slow path: at least one character must be replaced. Range over runes so
	// each invalid rune (which may be multi-byte) collapses to a single '_'.
	for _, r := range input {
		if !isValidCharacter(r) {
			sb.WriteRune('_')
		} else {
			sb.WriteRune(r)
		}
	}
}

// isCleanForProtocol reports whether input is composed entirely of valid
// characters and therefore requires no sanitization. It scans bytes rather than
// runes: any byte >= 0x80 (part of a multi-byte rune) fails isValidCharacter,
// correctly routing non-ASCII input to the rewriting slow path.
func isCleanForProtocol(input string) bool {
	for i := 0; i < len(input); i++ {
		if !isValidCharacter(rune(input[i])) {
			return false
		}
	}
	return true
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
