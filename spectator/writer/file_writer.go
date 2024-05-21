package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"os"
)

// FileWriter is a writer that writes to a file.
type FileWriter struct {
	file   *os.File
	logger logger.Logger
}

// NewFileWriter creates a new FileWriter.
func NewFileWriter(filename string, logger logger.Logger) (*FileWriter, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileWriter{file, logger}, nil
}

// Write writes a line to the file.
func (f *FileWriter) Write(line string) {
	f.logger.Debugf("Sending line: %s", line)

	_, err := fmt.Fprintln(f.file, line)
	if err != nil {
		f.logger.Errorf("Error writing to file: %s", err)
	}
}

// Close closes the file.
func (f *FileWriter) Close() error {
	return f.file.Close()
}
