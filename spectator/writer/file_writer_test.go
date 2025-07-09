package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"os"
	"strings"
	"testing"
)

const testFileName = "test.txt"

func TestNewFileWriter(t *testing.T) {
	defer os.Remove(testFileName)

	writer, err := NewFileWriter(testFileName, logger.NewDefaultLogger())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if writer == nil {
		t.Errorf("Expected writer to be not nil")
	}
}

func TestFileWriter_Write(t *testing.T) {
	defer os.Remove(testFileName)

	writer, _ := NewFileWriter(testFileName, logger.NewDefaultLogger())

	line := "test line"
	writer.Write(line)

	content, _ := os.ReadFile(testFileName)
	if strings.TrimRight(string(content), "\n") != line {
		t.Errorf("Expected '%s', got '%s'", line, string(content))
	}
}

func TestFileWriter_WriteBytes(t *testing.T) {
	defer os.Remove(testFileName)

	writer, _ := NewFileWriter(testFileName, logger.NewDefaultLogger())

	line := "test line"
	writer.WriteBytes([]byte(line))

	content, _ := os.ReadFile(testFileName)
	if strings.TrimRight(string(content), "\n") != line {
		t.Errorf("Expected '%s', got '%s'", line, string(content))
	}
}

func TestFileWriter_WriteString(t *testing.T) {
	defer os.Remove(testFileName)

	writer, _ := NewFileWriter(testFileName, logger.NewDefaultLogger())

	line := "test line"
	writer.WriteString(line)

	content, _ := os.ReadFile(testFileName)
	if strings.TrimRight(string(content), "\n") != line {
		t.Errorf("Expected '%s', got '%s'", line, string(content))
	}
}

// Test using a FileWriter with an existing file
func TestFileWriter_WriteExistingFile(t *testing.T) {
	defer os.Remove(testFileName)

	// Create a file with some content
	os.WriteFile(testFileName, []byte("existing content\n"), 0644)

	writer, _ := NewFileWriter(testFileName, logger.NewDefaultLogger())

	line := "test line"
	writer.Write(line)

	content, _ := os.ReadFile(testFileName)
	expected := "existing content\ntest line\n"
	if string(content) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(content))
	}

}

func TestFileWriter_Close(t *testing.T) {
	defer os.Remove(testFileName)

	writer, _ := NewFileWriter(testFileName, logger.NewDefaultLogger())
	err := writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
