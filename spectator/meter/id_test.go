package meter

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

func TestNewCounterDirect_MeterIdMatchesWithId(t *testing.T) {
	w := &writer.NoopWriter{}
	tags := map[string]string{"app": "foo", "region": "us-east-1", "env": "prod"}

	viaId := NewCounter(NewId("my.counter", tags), w)
	direct := NewCounterDirect("my.counter", tags, nil, w)

	// Canonical identity must match. MapKey sorts tags, so it is order-independent
	// (spectatordId tag order follows map iteration order and is not stable).
	got := direct.MeterId()
	if got.MapKey() != viaId.MeterId().MapKey() {
		t.Errorf("MapKey mismatch: %q vs %q", got.MapKey(), viaId.MeterId().MapKey())
	}

	// MeterId() reconstructed from the prefix must recover the name and (for
	// clean input) the original tags.
	if got.Name() != "my.counter" {
		t.Errorf("name mismatch: got %q", got.Name())
	}
	if !reflect.DeepEqual(got.Tags(), tags) {
		t.Errorf("tags mismatch: got %v want %v", got.Tags(), tags)
	}
}

func TestNewCounterDirect_MeterIdNoTags(t *testing.T) {
	direct := NewCounterDirect("my.counter", nil, nil, &writer.NoopWriter{})
	got := direct.MeterId()
	if got.Name() != "my.counter" {
		t.Errorf("name mismatch: got %q", got.Name())
	}
	if len(got.Tags()) != 0 {
		t.Errorf("expected no tags, got %v", got.Tags())
	}
}

func TestId_mapKey(t *testing.T) {
	id := NewId("foo", nil)
	k := id.MapKey()
	if k != "foo" {
		t.Error("Expected foo, got", k)
	}

	reusesKey := Id{
		name: "foo",
		key:  "bar",
	}
	k2 := reusesKey.MapKey()
	if k2 != "bar" {
		t.Error("Expected MapKey to be reused: bar !=", k2)
	}
}

func TestId_mapKeyConcurrent(t *testing.T) {
	id := NewId("foo", nil)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		_ = id.MapKey()
		wg.Done()
	}()
	go func() {
		_ = id.MapKey()
		wg.Done()
	}()

	wg.Wait()
}

func TestId_mapKeySortsTags(t *testing.T) {
	tags := map[string]string{}

	for i := 0; i < 100; i++ {
		k := fmt.Sprintf("%03d", i)
		tags[k] = "v"
	}
	id := NewId("foo", tags)

	var buf bytes.Buffer
	buf.WriteString("foo")
	for i := 0; i < 100; i++ {
		k := fmt.Sprintf("|%03d|v", i)
		buf.WriteString(k)
	}

	k := id.MapKey()
	if k != buf.String() {
		t.Errorf("Expected %s, got %s", buf.String(), k)
	}
}

func TestId_copiesTags(t *testing.T) {
	tags := map[string]string{"foo": "abc", "bar": "def"}
	id := NewId("foo", tags)

	tags["foo"] = "zzz"
	if id.Tags()["foo"] != "abc" {
		t.Errorf("Expected ids to create a copy of the tags. Got '%s', expected 'abc'", id.Tags()["foo"])
	}
}

func TestId_Accessors(t *testing.T) {
	id := NewId("foo", map[string]string{"foo": "abc", "bar": "def"})
	if id.Name() != "foo" {
		t.Errorf("Expected name=foo, got name=%s", id.Name())
	}

	expected := map[string]string{"foo": "abc", "bar": "def"}
	if !reflect.DeepEqual(expected, id.Tags()) {
		t.Errorf("Expected tags=%v, got %v", expected, id.Tags())
	}
}

func TestId_WithTags(t *testing.T) {
	id1 := NewId("c", map[string]string{"statistic": "baz", "a": "b"})
	id2 := id1.WithTags(map[string]string{"statistic": "foo", "k": "v"})
	expected := map[string]string{"statistic": "foo", "k": "v", "a": "b"}
	if id2.Name() != "c" {
		t.Errorf("WithTags must copy the name. Got %s instead of c", id2.Name())
	}

	if !reflect.DeepEqual(expected, id2.Tags()) {
		t.Errorf("Expected %v, got %v tags", expected, id2.Tags())
	}
}

func TestToSpectatorId(t *testing.T) {
	name := "test"
	tags := map[string]string{
		"tag1": "value1",
		"tag2": "value2",
	}

	// The order of the tags is not guaranteed
	expected1 := "test,tag1=value1,tag2=value2"
	expected2 := "test,tag2=value2,tag1=value1"
	result := toSpectatorId(name, tags)

	if result != expected1 && result != expected2 {
		t.Errorf("Expected '%s' or '%s', got '%s'", expected1, expected2, result)
	}
}

func TestToSpectatorId_EmptyTags(t *testing.T) {
	name := "test"
	tags := map[string]string{}

	expected := "test"
	result := toSpectatorId(name, tags)

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestToSpectatorId_InvalidTags(t *testing.T) {
	name := "test`!@#$%^&*()-=~_+[]{}\\|;:'\",<.>/?foo"
	tags := map[string]string{
		"tag1,:=": "value1,:=",
		"tag2,;=": "value2,;=",
	}

	expected1 := "test______^____-_~______________.___foo,tag1___=value1___,tag2___=value2___"
	expected2 := "test______^____-_~______________.___foo,tag2___=value2___,tag1___=value1___"
	result := toSpectatorId(name, tags)

	if result != expected1 && result != expected2 {
		t.Errorf("Expected '%s' or '%s', got '%s'", expected1, expected2, result)
	}
}

// TestToSpectatorId_MultiByte verifies that multi-byte (non-ASCII) runes each
// collapse to a single '_', which exercises the writeSanitized slow path and
// confirms the byte-scan fast path correctly routes non-ASCII input to it.
func TestToSpectatorId_MultiByte(t *testing.T) {
	// "café" -> c,a,f valid; 'é' (2 bytes) invalid -> one '_'. "€5" -> '€' (3 bytes) -> '_', '5' valid.
	result := toSpectatorId("café", map[string]string{"k": "€5"})
	if expected := "caf_,k=_5"; result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

var benchName = "my.metric.with_a_fairly_long_name.and.some.invalid.chars!@#"
var benchTags = map[string]string{
	"tag1":         "value1",
	"another_tag":  "another_value_with_some_length",
	"invalid-key!": "invalid-value@",
	"tag4":         "value4",
	"last.tag":     "final~value",
}

func BenchmarkToSpectatorId(b *testing.B) {
	replaceInvalidCharacters := func(input string) string {
		var result strings.Builder
		for _, r := range input {
			if !isValidCharacter(r) {
				result.WriteRune('_')
			} else {
				result.WriteRune(r)
			}
		}
		return result.String()

	}
	originalToSpectatorId := func(name string, tags map[string]string) string {
		result := replaceInvalidCharacters(name)

		for k, v := range tags {
			k = replaceInvalidCharacters(k)
			v = replaceInvalidCharacters(v)
			result += fmt.Sprintf(",%s=%s", k, v)

		}

		return result
	}

	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		_ = originalToSpectatorId(benchName, benchTags)
	}
}

func BenchmarkToSpectatorIdBuilder(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		_ = toSpectatorId(benchName, benchTags)
	}
}
