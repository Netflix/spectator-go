package writer

import (
	"slices"
	"sync"
)

// MemoryWriter stores lines in memory in an array, so updates can be inspected for test validation.
type MemoryWriter struct {
	lines []string
	mu    sync.RWMutex
}

func (m *MemoryWriter) Write(line string) {
	m.WriteString(line)
}

func (m *MemoryWriter) WriteLine(prefix, value string) {
	m.WriteString(prefix + value)
}

func (m *MemoryWriter) WriteInt(prefix string, value int64) {
	m.WriteString(formatLineInt(prefix, value))
}

func (m *MemoryWriter) WriteUint(prefix string, value uint64) {
	m.WriteString(formatLineUint(prefix, value))
}

func (m *MemoryWriter) WriteFloat(prefix string, value float64) {
	m.WriteString(formatLineFloat(prefix, value))
}

func (m *MemoryWriter) WriteBytes(line []byte) {
	m.WriteString(string(line))
}

func (m *MemoryWriter) WriteString(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lines = append(m.lines, line)
}

func (m *MemoryWriter) Lines() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.lines)
}

func (m *MemoryWriter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lines = []string{}
}

func (m *MemoryWriter) Close() error {
	return nil
}
