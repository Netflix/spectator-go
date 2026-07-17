package writer

// NoopWriter is a writer that does nothing.
type NoopWriter struct{}

func (n *NoopWriter) Write(_ string) {}

func (n *NoopWriter) WriteLine(_, _ string) {}

func (n *NoopWriter) WriteInt(_ string, _ int64) {}

func (n *NoopWriter) WriteUint(_ string, _ uint64) {}

func (n *NoopWriter) WriteFloat(_ string, _ float64) {}

func (n *NoopWriter) WriteBytes(_ []byte) {}

func (n *NoopWriter) WriteString(_ string) {}

func (n *NoopWriter) Close() error {
	return nil
}
