package meter

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

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

func TestId_WithTag(t *testing.T) {
	tests := map[string]struct {
		baseTags     map[string]string
		key          string
		value        string
		expectedTags map[string]string
	}{
		"add new key": {
			baseTags:     map[string]string{"a": "1"},
			key:          "b",
			value:        "2",
			expectedTags: map[string]string{"a": "1", "b": "2"},
		},
		"replace existing key": {
			baseTags:     map[string]string{"a": "1", "b": "2"},
			key:          "a",
			value:        "replaced",
			expectedTags: map[string]string{"a": "replaced", "b": "2"},
		},
		"no existing tags": {
			baseTags:     nil,
			key:          "a",
			value:        "1",
			expectedTags: map[string]string{"a": "1"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			id := NewId("foo", tt.baseTags)
			id2 := id.WithTag(tt.key, tt.value)

			if !reflect.DeepEqual(tt.expectedTags, id2.Tags()) {
				t.Errorf("Expected %v, got %v", tt.expectedTags, id2.Tags())
			}
		})
	}
}

func TestId_WithTag_DoesNotMutateOriginal(t *testing.T) {
	id := NewId("foo", map[string]string{"a": "1"})
	_ = id.WithTag("b", "2")

	expected := map[string]string{"a": "1"}
	if !reflect.DeepEqual(expected, id.Tags()) {
		t.Errorf("Original mutated: expected %v, got %v", expected, id.Tags())
	}
}

func TestToSpectatorIdFromFlat_InvalidTags(t *testing.T) {
	name := "test`!@#$%^&*()-=~_+[]{}\\|;:'\",<.>/?foo"
	flat := []string{"tag1,:=", "value1,:=", "tag2,;=", "value2,;="}
	result := toSpectatorIdFromFlat(name, flat)

	// Tags appear in insertion order (flat slice), so order is deterministic.
	expected := "test______^____-_~______________.___foo,tag1___=value1___,tag2___=value2___"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestToSpectatorIdFromFlat(t *testing.T) {
	tests := map[string]struct {
		metric   string
		flatTags []string
		expected string
	}{
		"no tags":        {metric: "test", flatTags: nil, expected: "test"},
		"empty tags":     {metric: "test", flatTags: []string{}, expected: "test"},
		"one tag":        {metric: "test", flatTags: []string{"k1", "v1"}, expected: "test,k1=v1"},
		"two tags":       {metric: "test", flatTags: []string{"k1", "v1", "k2", "v2"}, expected: "test,k1=v1,k2=v2"},
		"sanitized name": {metric: "test!@#", flatTags: nil, expected: "test___"},
		"sanitized tags": {metric: "test", flatTags: []string{"k!ey", "v@lue"}, expected: "test,k_ey=v_lue"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := toSpectatorIdFromFlat(tt.metric, tt.flatTags)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
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

func BenchmarkToSpectatorIdFromFlat(b *testing.B) {
	flat := make([]string, 0, 2*len(benchTags))
	for k, v := range benchTags {
		flat = append(flat, k, v)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = toSpectatorIdFromFlat(benchName, flat)
	}
}
