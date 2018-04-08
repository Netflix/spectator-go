package spectator

import (
	"bytes"
	"fmt"
	"testing"
)

func TestId_mapKey(t *testing.T) {
	id := newId("foo", nil)
	k := id.mapKey()
	if k != "foo" {
		t.Error("Expected foo, got", k)
	}

	reusesKey := Id{"foo", nil, "bar"}
	k2 := reusesKey.mapKey()
	if k2 != "bar" {
		t.Error("Expected mapKey to be reused: bar !=", k2)
	}
}

func TestId_mapKeySortsTags(t *testing.T) {
	tags := make(map[string]string)

	for i := 0; i < 100; i++ {
		k := fmt.Sprintf("%03d", i)
		tags[k] = "v"
	}
	id := newId("foo", tags)
	k := id.mapKey()

	var buf bytes.Buffer
	buf.WriteString("foo")
	for i := 0; i < 100; i++ {
		k := fmt.Sprintf("|%03d|v", i)
		buf.WriteString(k)
	}

	if k != buf.String() {
		t.Errorf("Expected %s, got %s", buf.String(), k)
	}
}
