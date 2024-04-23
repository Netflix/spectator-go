package writer

import "fmt"

// Writer that accepts SpectatorD line protocol.
type Writer interface {
	Write(line string) // TODO check type
}

// TODO consider extracting common logging logic into a default trait that is then included in all implementations

type PrintWriter struct {
}

func (p *PrintWriter) Write(line string) {
	fmt.Print(line)
}

type MemoryWriter struct {
	Lines []string
}

func (m *MemoryWriter) Write(line string) {
	m.Lines = append(m.Lines, line)
}
