package writer

// NoopWriter is a writer that does nothing.
type NoopWriter struct{}

func (n *NoopWriter) Write(_ string) {}

func (n *NoopWriter) WriteBytes(_ []byte) {}

func (n *NoopWriter) WriteString(_ string) {}

func (n *NoopWriter) Close() error {
	return nil
}
