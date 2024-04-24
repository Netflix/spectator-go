package writer

import (
	"github.com/Netflix/spectator-go/spectator/logger"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestNewFileWriter(t *testing.T) {
	defer os.Remove("testfile.txt")

	writer, err := NewFileWriter("testfile.txt", logger.NewDefaultLogger())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if writer == nil {
		t.Errorf("Expected writer to be not nil")
	}
}

func TestFileWriter_Write(t *testing.T) {
	defer os.Remove("testfile.txt")

	writer, _ := NewFileWriter("testfile.txt", logger.NewDefaultLogger())

	line := "test line"
	writer.Write(line)

	content, _ := ioutil.ReadFile("testfile.txt")
	if strings.TrimRight(string(content), "\n") != line {
		t.Errorf("Expected '%s', got '%s'", line, string(content))
	}
}

func TestFileWriter_Close(t *testing.T) {
	defer os.Remove("testfile.txt")

	writer, _ := NewFileWriter("testfile.txt", logger.NewDefaultLogger())
	err := writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
