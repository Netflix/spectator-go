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

func (f *FileWriter) WriteLine(prefix, value string) {
	f.WriteString(prefix + value)
}

func (f *FileWriter) WriteInt(prefix string, value int64) {
	f.WriteString(formatLineInt(prefix, value))
}

func (f *FileWriter) WriteUint(prefix string, value uint64) {
	f.WriteString(formatLineUint(prefix, value))
}

func (f *FileWriter) WriteFloat(prefix string, value float64) {
	f.WriteString(formatLineFloat(prefix, value))
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
