package writer

import (
	"github.com/Netflix/spectator-go/spectator/logger"
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

	content, _ := os.ReadFile("testfile.txt")
	if strings.TrimRight(string(content), "\n") != line {
		t.Errorf("Expected '%s', got '%s'", line, string(content))
	}
}

// Test using a FileWriter with an existing file
func TestFileWriter_WriteExistingFile(t *testing.T) {
	defer os.Remove("testfile.txt")

	// Create a file with some content
	os.WriteFile("testfile.txt", []byte("existing content\n"), 0644)

	writer, _ := NewFileWriter("testfile.txt", logger.NewDefaultLogger())

	line := "test line"
	writer.Write(line)

	content, _ := os.ReadFile("testfile.txt")
	expected := "existing content\ntest line\n"
	if string(content) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(content))
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
