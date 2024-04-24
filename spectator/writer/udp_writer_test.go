package writer

import (
	"github.com/Netflix/spectator-go/spectator/logger"
	"net"
	"testing"
	"time"
)

func TestNewUdpWriter(t *testing.T) {
	writer, err := NewUdpWriter("localhost:5000", logger.NewDefaultLogger())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if writer == nil {
		t.Errorf("Expected writer to be not nil")
	}
}

func TestUdpWriter_Close(t *testing.T) {
	writer, _ := NewUdpWriter("localhost:5000", logger.NewDefaultLogger())
	err := writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewUdpWriter_InvalidAddress(t *testing.T) {
	writer, err := NewUdpWriter("invalid address", logger.NewDefaultLogger())
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if writer != nil {
		t.Errorf("Expected writer to be nil")
	}
}

// Test write after close using a local UDP server
func TestUdpWriter_WriteAfterClose(t *testing.T) {
	// Start a local UDP server
	pc, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("Could not start UDP server: %v", err)
	}
	defer pc.Close()

	// Create a new UDP writer
	writer, err := NewUdpWriter(pc.LocalAddr().String(), logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Could not create UDP writer: %v", err)
	}

	// Close the writer
	_ = writer.Close()

	// Write a message
	writer.Write("test message")

	// Check that no message was received
	buffer := make([]byte, 1024)
	_ = pc.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
	_, _, err = pc.ReadFrom(buffer)
	// ReadFrom will throw error if no message is received after timeout
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

}

func TestUdpWriter_Write(t *testing.T) {
	// Start a local UDP server
	pc, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("Could not start UDP server: %v", err)
	}
	defer pc.Close()

	// Create a new UDP writer
	writer, err := NewUdpWriter(pc.LocalAddr().String(), logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Could not create UDP writer: %v", err)
	}

	// Write a message
	message := "test message"
	writer.Write(message)

	// Read the message from the UDP server
	buffer := make([]byte, len(message))
	_ = pc.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
	n, _, err := pc.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Could not read from UDP server: %v", err)
	}

	// Check the message
	if string(buffer[:n]) != message {
		t.Errorf("Expected '%s', got '%s'", message, string(buffer[:n]))
	}
}
