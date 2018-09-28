package spectator

import (
	"bytes"
	"fmt"
	"sort"
)

type Id struct {
	name string
	tags map[string]string
	key  string
}

// computes and saves a key to be used to address Ids in maps
func (id *Id) mapKey() string {
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

func newId(name string, tags map[string]string) *Id {
	if tags == nil {
		tags = map[string]string{}
	}
	return &Id{name, tags, ""}
}

func (id *Id) WithTag(key string, value string) *Id {
	newTags := make(map[string]string)

	for k, v := range id.tags {
		newTags[k] = v
	}
	newTags[key] = value

	return newId(id.name, newTags)
}

func (id *Id) WithStat(stat string) *Id {
	return id.WithTag("statistic", stat)
}

func (id *Id) String() string {
	return fmt.Sprintf("Id{name=%s,tags=%v}", id.name, id.tags)
}
