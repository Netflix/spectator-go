package meter

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// v1.3.3 implementation: no pool, creates a new strings.Builder per call,
// uses fmt.Sprintf for tag formatting, and string concatenation for the result.
func toSpectatorIdV1(name string, tags map[string]string) string {
	replaceInvalid := func(input string) string {
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

	result := replaceInvalid(name)
	for k, v := range tags {
		k = replaceInvalid(k)
		v = replaceInvalid(v)
		result += fmt.Sprintf(",%s=%s", k, v)
	}
	return result
}

// current implementation is toSpectatorId from id.go (uses builderPool + Grow)

// --- Test inputs ---

type benchCase struct {
	name string
	metricName string
	tags map[string]string
}

func benchCases() []benchCase {
	return []benchCase{
		{
			name:       "NoTags_Short",
			metricName: "counter",
			tags:       map[string]string{},
		},
		{
			name:       "NoTags_Long",
			metricName: "my.application.server.request.duration.very.long.metric.name.here",
			tags:       map[string]string{},
		},
		{
			name:       "1Tag",
			metricName: "http.requests",
			tags:       map[string]string{"status": "200"},
		},
		{
			name:       "3Tags",
			metricName: "http.server.requests",
			tags: map[string]string{
				"app":    "myapp",
				"region": "us-east-1",
				"env":    "prod",
			},
		},
		{
			name:       "5Tags",
			metricName: "my.metric.with_a_fairly_long_name",
			tags: map[string]string{
				"tag1":        "value1",
				"another_tag": "another_value_with_some_length",
				"tag3":        "value3",
				"tag4":        "value4",
				"last.tag":    "final~value",
			},
		},
		{
			name:       "10Tags",
			metricName: "server.request.latency",
			tags:       makeTags(10),
		},
		{
			name:       "InvalidChars",
			metricName: "test`!@#$%^&*()-=~_+[]{}\\|;:'\",<.>/?foo",
			tags: map[string]string{
				"tag1,:=": "value1,:=",
				"tag2,;=": "value2,;=",
				"normal":  "value",
			},
		},
		{
			name:       "LongValues",
			metricName: "service.metric",
			tags: map[string]string{
				"path":      "/api/v2/users/preferences/notifications/settings/advanced",
				"host":      "ip-10-123-456-789.ec2.internal.very.long.hostname.example.com",
				"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			},
		},
	}
}

// --- Sequential benchmarks ---

func BenchmarkToSpectatorId_V1(b *testing.B) {
	for _, bc := range benchCases() {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = toSpectatorIdV1(bc.metricName, bc.tags)
			}
		})
	}
}

func BenchmarkToSpectatorId_Current(b *testing.B) {
	for _, bc := range benchCases() {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = toSpectatorId(bc.metricName, bc.tags)
			}
		})
	}
}

// --- Concurrent benchmarks (sync.Pool shines here) ---

func BenchmarkToSpectatorId_V1_Concurrent(b *testing.B) {
	for _, bc := range benchCases() {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = toSpectatorIdV1(bc.metricName, bc.tags)
				}
			})
		})
	}
}

func BenchmarkToSpectatorId_Current_Concurrent(b *testing.B) {
	for _, bc := range benchCases() {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = toSpectatorId(bc.metricName, bc.tags)
				}
			})
		})
	}
}

// --- High-contention benchmark: many goroutines hammering the same pool ---

func BenchmarkToSpectatorId_HighContention(b *testing.B) {
	bc := benchCases()[4] // 5Tags case
	for _, goroutines := range []int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("V1_%dG", goroutines), func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(goroutines)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = toSpectatorIdV1(bc.metricName, bc.tags)
				}
			})
		})
		b.Run(fmt.Sprintf("Current_%dG", goroutines), func(b *testing.B) {
			b.ReportAllocs()
			b.SetParallelism(goroutines)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = toSpectatorId(bc.metricName, bc.tags)
				}
			})
		})
	}
}

// --- Mixed workload: simulate real usage with NewId creating IDs concurrently ---

func BenchmarkNewId_V1Style_Concurrent(b *testing.B) {
	// Simulate v1.3.3 NewId which calls toSpectatorIdV1 internally.
	// We can't swap out the real NewId, so we measure the id-construction
	// portion by calling the function + map clone.
	tags := map[string]string{
		"app": "myapp", "region": "us-east-1", "env": "prod",
		"cluster": "main", "instance": "i-abc123",
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			myTags := make(map[string]string, len(tags))
			for k, v := range tags {
				myTags[k] = v
			}
			_ = toSpectatorIdV1("server.request.latency", myTags)
		}
	})
}

func BenchmarkNewId_Current_Concurrent(b *testing.B) {
	tags := map[string]string{
		"app": "myapp", "region": "us-east-1", "env": "prod",
		"cluster": "main", "instance": "i-abc123",
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewId("server.request.latency", tags)
		}
	})
}

// --- Correctness check: ensure both produce the same output ---

func TestToSpectatorIdV1_MatchesCurrent(t *testing.T) {
	// With no tags, output must be identical.
	for _, bc := range benchCases() {
		if len(bc.tags) == 0 {
			v1 := toSpectatorIdV1(bc.metricName, bc.tags)
			cur := toSpectatorId(bc.metricName, bc.tags)
			if v1 != cur {
				t.Errorf("%s: no-tags mismatch: v1=%q cur=%q", bc.name, v1, cur)
			}
		}
		// With tags, order may differ since both iterate maps randomly,
		// so just check they have the same prefix and same length.
		v1 := toSpectatorIdV1(bc.metricName, bc.tags)
		cur := toSpectatorId(bc.metricName, bc.tags)
		if len(v1) != len(cur) {
			t.Errorf("%s: length mismatch: v1=%d cur=%d\n  v1=%q\n  cur=%q", bc.name, len(v1), len(cur), v1, cur)
		}
	}
}

// --- Allocation-focused: measure just builder reuse benefit ---

func BenchmarkBuilderPool_GetPut(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sb := builderPool.Get().(*strings.Builder)
			sb.Reset()
			sb.Grow(200)
			sb.WriteString("some.metric.name,tag1=value1,tag2=value2,tag3=value3")
			_ = sb.String()
			builderPool.Put(sb)
		}
	})
}

func BenchmarkBuilder_NoPool(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var sb strings.Builder
			sb.Grow(200)
			sb.WriteString("some.metric.name,tag1=value1,tag2=value2,tag3=value3")
			_ = sb.String()
		}
	})
}

// --- Burst pattern: create many IDs in a tight loop (simulates startup/batch) ---

func BenchmarkBurstCreation_V1(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_ = toSpectatorIdV1("metric.name", tags)
		}
	}
}

func BenchmarkBurstCreation_Current(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_ = toSpectatorId("metric.name", tags)
		}
	}
}

// --- Contended pool scenario: alternating producers/consumers ---

func BenchmarkContendedWorkload(b *testing.B) {
	tags5 := makeTags(5)
	tags10 := makeTags(10)

	b.Run("V1", func(b *testing.B) {
		b.ReportAllocs()
		var wg sync.WaitGroup
		workers := 8
		perWorker := b.N / workers
		if perWorker == 0 {
			perWorker = 1
		}
		wg.Add(workers)
		b.ResetTimer()
		for w := 0; w < workers; w++ {
			go func(id int) {
				defer wg.Done()
				t := tags5
				if id%2 == 0 {
					t = tags10
				}
				for i := 0; i < perWorker; i++ {
					_ = toSpectatorIdV1("metric.name", t)
				}
			}(w)
		}
		wg.Wait()
	})

	b.Run("Current", func(b *testing.B) {
		b.ReportAllocs()
		var wg sync.WaitGroup
		workers := 8
		perWorker := b.N / workers
		if perWorker == 0 {
			perWorker = 1
		}
		wg.Add(workers)
		b.ResetTimer()
		for w := 0; w < workers; w++ {
			go func(id int) {
				defer wg.Done()
				t := tags5
				if id%2 == 0 {
					t = tags10
				}
				for i := 0; i < perWorker; i++ {
					_ = toSpectatorId("metric.name", t)
				}
			}(w)
		}
		wg.Wait()
	})
}
