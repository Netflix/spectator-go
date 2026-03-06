package meter

import (
	"fmt"
	"testing"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

var noopWriter = &writer.NoopWriter{}

// Pre-generate varying tag values to avoid measuring Sprintf in benchmark loops.
var varyingValues [1024]string

func init() {
	for i := range varyingValues {
		varyingValues[i] = fmt.Sprintf("value-%d", i)
	}
}

func makeTags(n int) map[string]string {
	tags := make(map[string]string, n)
	for i := 0; i < n; i++ {
		tags[fmt.Sprintf("key%d", i)] = fmt.Sprintf("val%d", i)
	}
	return tags
}

// --- Id creation benchmarks ---

func BenchmarkNewId(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewId("test.metric.name", tags)
	}
}

func BenchmarkNewId_ManyTags(b *testing.B) {
	tags := makeTags(10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewId("test.metric.name", tags)
	}
}

func BenchmarkWithTag(b *testing.B) {
	baseId := NewId("test.metric.name", map[string]string{"app": "myapp", "region": "us-east-1"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		baseId.WithTag("instance", varyingValues[i&1023])
	}
}

func BenchmarkWithTags(b *testing.B) {
	baseId := NewId("test.metric.name", map[string]string{"app": "myapp"})
	extraTags := map[string]string{"region": "us-east-1", "env": "prod", "cluster": "main"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		baseId.WithTags(extraTags)
	}
}

// --- Tag count scaling ---

func BenchmarkNewId_TagScaling(b *testing.B) {
	for _, n := range []int{0, 1, 3, 5, 10} {
		tags := makeTags(n)
		b.Run(fmt.Sprintf("tags-%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				NewId("test.metric.name", tags)
			}
		})
	}
}

// --- Counter benchmarks ---

func BenchmarkNewCounter(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := NewId("test.counter", tags)
		NewCounter(id, noopWriter)
	}
}

func BenchmarkCounter_Increment(b *testing.B) {
	id := NewId("test.counter", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	c := NewCounter(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Increment()
	}
}

func BenchmarkCounter_Add(b *testing.B) {
	id := NewId("test.counter", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	c := NewCounter(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Add(42)
	}
}

func BenchmarkCounter_VaryingTags(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tags := map[string]string{"app": "myapp", "instance": varyingValues[i&1023]}
		id := NewId("test.counter", tags)
		c := NewCounter(id, noopWriter)
		c.Increment()
	}
}

// --- Gauge benchmarks ---

func BenchmarkNewGauge(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := NewId("test.gauge", tags)
		NewGauge(id, noopWriter)
	}
}

func BenchmarkGauge_Set(b *testing.B) {
	id := NewId("test.gauge", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	g := NewGauge(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Set(3.14159)
	}
}

func BenchmarkGauge_SetInt(b *testing.B) {
	id := NewId("test.gauge", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	g := NewGauge(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.SetInt(42)
	}
}

func BenchmarkGauge_Set_IntValue(b *testing.B) {
	id := NewId("test.gauge", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	g := NewGauge(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Set(float64(42))
	}
}

func BenchmarkGauge_VaryingTags(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tags := map[string]string{"app": "myapp", "instance": varyingValues[i&1023]}
		id := NewId("test.gauge", tags)
		g := NewGauge(id, noopWriter)
		g.Set(3.14159)
	}
}

// --- DistributionSummary benchmarks ---

func BenchmarkNewDistributionSummary(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := NewId("test.distsummary", tags)
		NewDistributionSummary(id, noopWriter)
	}
}

func BenchmarkDistributionSummary_Record(b *testing.B) {
	id := NewId("test.distsummary", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	d := NewDistributionSummary(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Record(int64(i))
	}
}

func BenchmarkDistributionSummary_VaryingTags(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tags := map[string]string{"app": "myapp", "instance": varyingValues[i&1023]}
		id := NewId("test.distsummary", tags)
		d := NewDistributionSummary(id, noopWriter)
		d.Record(int64(i))
	}
}

// --- Timer benchmarks ---

func BenchmarkNewTimer(b *testing.B) {
	tags := map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := NewId("test.timer", tags)
		NewTimer(id, noopWriter)
	}
}

func BenchmarkTimer_Record(b *testing.B) {
	id := NewId("test.timer", map[string]string{"app": "myapp", "region": "us-east-1", "env": "prod"})
	t := NewTimer(id, noopWriter)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.Record(time.Duration(i) * time.Millisecond)
	}
}

func BenchmarkTimer_VaryingTags(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tags := map[string]string{"app": "myapp", "instance": varyingValues[i&1023]}
		id := NewId("test.timer", tags)
		t := NewTimer(id, noopWriter)
		t.Record(time.Duration(i) * time.Millisecond)
	}
}
