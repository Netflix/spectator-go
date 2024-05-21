package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"os"
	"testing"
)

func TestValidOutputLocation(t *testing.T) {
	testCases := []struct {
		outputLocation string
		expected       bool
	}{
		{"none", true},
		{"memory", true},
		{"stdout", true},
		{"stderr", true},
		{"file://testfile.txt", true},
		{"udp://localhost:1234", true},
		{"invalid", false},
	}

	for _, tc := range testCases {
		result := IsValidOutputLocation(tc.outputLocation)
		if result != tc.expected {
			t.Errorf("Expected %v for output location '%s', got %v", tc.expected, tc.outputLocation, result)
		}
	}
}

func TestNewWriter(t *testing.T) {
	testCases := []struct {
		outputLocation string
		expectedType   string
	}{
		{"none", "*writer.NoopWriter"},
		{"memory", "*writer.MemoryWriter"},
		{"stdout", "*writer.StdoutWriter"},
		{"stderr", "*writer.StderrWriter"},
		{"file://testfile.txt", "*writer.FileWriter"},
		{"udp://localhost:5000", "*writer.UdpWriter"},
	}

	for _, tc := range testCases {
		writer, _ := NewWriter(tc.outputLocation, logger.NewDefaultLogger())
		resultType := fmt.Sprintf("%T", writer)
		if resultType != tc.expectedType {
			t.Errorf("Expected %s for output location '%s', got %s", tc.expectedType, tc.outputLocation, resultType)
		}

		// Cleanup test file
		_ = os.Remove("testfile.txt")
	}
}

func TestNewWriter_InvalidOutputLocation(t *testing.T) {
	_, err := NewWriter("invalid", logger.NewDefaultLogger())
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestNewWriter_EmptyOutputLocation(t *testing.T) {
	_, err := NewWriter("", logger.NewDefaultLogger())
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
