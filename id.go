package spectator

import (
	"crypto/md5"
	"fmt"
	"math/big"
	"sort"
)

type Id struct {
	name string
	tags map[string]string
	hash uint64
}

func (id *Id) Hash() uint64 {
	if id.hash != 0 {
		return id.hash
	}

	hasher := md5.New()
	hasher.Write([]byte(id.name))
	var keys []string
	for k := range id.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := id.tags[k]
		hasher.Write([]byte("|"))
		hasher.Write([]byte(k))
		hasher.Write([]byte("|"))
		hasher.Write([]byte(v))
	}
	res := big.NewInt(0)
	res.SetBytes(hasher.Sum(nil)[:8])
	id.hash = res.Uint64()
	return id.hash
}

func newId(name string, tags map[string]string) *Id {
	if tags == nil {
		tags = map[string]string{}
	}
	return &Id{name, tags, 0}
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
