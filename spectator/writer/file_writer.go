package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"os"
)

type FileWriter struct {
	file   *os.File
	logger logger.Logger
}

func NewFileWriter(filename string, logger logger.Logger) (*FileWriter, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileWriter{file, logger}, nil
}

func (f *FileWriter) Write(line string) {
	f.logger.Debugf("Sending line: %s", line)
	f.WriteString(line)
}

func (f *FileWriter) WriteBytes(line []byte) {
	f.WriteString(string(line))
}

func (f *FileWriter) WriteString(line string) {
	_, err := fmt.Fprintln(f.file, line)
	if err != nil {
		f.logger.Errorf("Error writing to file: %s", err)
	}
}

func (f *FileWriter) Close() error {
	return f.file.Close()
}
