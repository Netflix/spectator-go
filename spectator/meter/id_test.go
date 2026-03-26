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

func TestTagPairs_clone(t *testing.T) {
	tests := map[string]struct {
		input      tagPairs
		extraPairs int
		wantLen    int
		wantCap    int
	}{
		"nil":            {input: nil, extraPairs: 0, wantLen: 0, wantCap: 0},
		"empty":          {input: tagPairs{}, extraPairs: 0, wantLen: 0, wantCap: 0},
		"extra capacity": {input: tagPairs{{key: "a", value: "1"}}, extraPairs: 3, wantLen: 1, wantCap: 4},
		"zero extra":     {input: tagPairs{{key: "a", value: "1"}}, extraPairs: 0, wantLen: 1, wantCap: 1},
		"multiple":       {input: tagPairs{{key: "a", value: "1"}, {key: "b", value: "2"}}, extraPairs: 1, wantLen: 2, wantCap: 3},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cloned := tt.input.clone(tt.extraPairs)
			if len(cloned) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(cloned), tt.wantLen)
			}
			if cap(cloned) != tt.wantCap {
				t.Errorf("cap = %d, want %d", cap(cloned), tt.wantCap)
			}
		})
	}
}

func TestTagPairs_clone_independent(t *testing.T) {
	orig := tagPairs{{key: "a", value: "1"}}
	cloned := orig.clone(0)
	cloned[0].value = "changed"
	if orig[0].value != "1" {
		t.Errorf("clone mutated original: got %q, want %q", orig[0].value, "1")
	}
}

func TestTagPairs_upsert(t *testing.T) {
	tests := map[string]struct {
		input    tagPairs
		key      string
		value    string
		wantLen  int
		wantTags map[string]string
	}{
		"nil append": {
			input: nil, key: "a", value: "1",
			wantLen: 1, wantTags: map[string]string{"a": "1"},
		},
		"empty append": {
			input: tagPairs{}, key: "a", value: "1",
			wantLen: 1, wantTags: map[string]string{"a": "1"},
		},
		"append new": {
			input: tagPairs{{key: "a", value: "1"}}, key: "b", value: "2",
			wantLen: 2, wantTags: map[string]string{"a": "1", "b": "2"},
		},
		"update existing": {
			input: tagPairs{{key: "a", value: "1"}, {key: "b", value: "2"}}, key: "a", value: "replaced",
			wantLen: 2, wantTags: map[string]string{"a": "replaced", "b": "2"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.input.upsert(tt.key, tt.value)
			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result), tt.wantLen)
			}
			if got := result.toMap(); !reflect.DeepEqual(got, tt.wantTags) {
				t.Errorf("toMap = %v, want %v", got, tt.wantTags)
			}
		})
	}
}

func TestTagPairs_sorted(t *testing.T) {
	tests := map[string]struct {
		input tagPairs
		want  []tagPair
	}{
		"nil":      {input: nil, want: []tagPair{}},
		"empty":    {input: tagPairs{}, want: []tagPair{}},
		"single":   {input: tagPairs{{key: "a", value: "1"}}, want: []tagPair{{key: "a", value: "1"}}},
		"reversed": {input: tagPairs{{key: "c", value: "3"}, {key: "a", value: "1"}, {key: "b", value: "2"}}, want: []tagPair{{key: "a", value: "1"}, {key: "b", value: "2"}, {key: "c", value: "3"}}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.input.sorted()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sorted = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagPairs_sorted_does_not_mutate(t *testing.T) {
	orig := tagPairs{{key: "c", value: "3"}, {key: "a", value: "1"}}
	_ = orig.sorted()
	if orig[0].key != "c" {
		t.Errorf("sorted mutated original: first key = %q, want %q", orig[0].key, "c")
	}
}

func TestTagPairs_toMap(t *testing.T) {
	tests := map[string]struct {
		input tagPairs
		want  map[string]string
	}{
		"nil":      {input: nil, want: map[string]string{}},
		"empty":    {input: tagPairs{}, want: map[string]string{}},
		"single":   {input: tagPairs{{key: "a", value: "1"}}, want: map[string]string{"a": "1"}},
		"multiple": {input: tagPairs{{key: "a", value: "1"}, {key: "b", value: "2"}}, want: map[string]string{"a": "1", "b": "2"}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.input.toMap()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toMap = %v, want %v", got, tt.want)
			}
		})
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

func TestToSpectatorIdFromPairs_InvalidTags(t *testing.T) {
	name := "test`!@#$%^&*()-=~_+[]{}\\|;:'\",<.>/?foo"
	tags := tagPairs{
		{key: "tag1,:=", value: "value1,:="},
		{key: "tag2,;=", value: "value2,;="},
	}
	result := toSpectatorIdFromPairs(name, tags)

	// Tags appear in insertion order, so order is deterministic.
	expected := "test______^____-_~______________.___foo,tag1___=value1___,tag2___=value2___"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestToSpectatorIdFromPairs(t *testing.T) {
	tests := map[string]struct {
		metric   string
		tags     tagPairs
		expected string
	}{
		"no tags":        {metric: "test", tags: nil, expected: "test"},
		"empty tags":     {metric: "test", tags: tagPairs{}, expected: "test"},
		"one tag":        {metric: "test", tags: tagPairs{{key: "k1", value: "v1"}}, expected: "test,k1=v1"},
		"two tags":       {metric: "test", tags: tagPairs{{key: "k1", value: "v1"}, {key: "k2", value: "v2"}}, expected: "test,k1=v1,k2=v2"},
		"sanitized name": {metric: "test!@#", tags: nil, expected: "test___"},
		"sanitized tags": {metric: "test", tags: tagPairs{{key: "k!ey", value: "v@lue"}}, expected: "test,k_ey=v_lue"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := toSpectatorIdFromPairs(tt.metric, tt.tags)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

var benchSinkString string

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
		benchSinkString = originalToSpectatorId(benchName, benchTags)
	}
}

func BenchmarkToSpectatorIdFromPairs(b *testing.B) {
	tags := make(tagPairs, 0, len(benchTags))
	for k, v := range benchTags {
		tags = append(tags, tagPair{key: k, value: v})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		benchSinkString = toSpectatorIdFromPairs(benchName, tags)
	}
}
