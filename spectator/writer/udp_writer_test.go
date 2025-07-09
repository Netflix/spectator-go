package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"net"
	"sort"
	"strconv"
	"sync"
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

func TestUdpWriter_NoBuffer(t *testing.T) {
	writer, _ := NewUdpWriterWithBuffer("localhost:5000", logger.NewDefaultLogger(), 0, time.Second)
	if writer.lineBuffer != nil {
		t.Errorf("Expected nil LineBuffer")
	}
	if writer.lowLatencyBuffer != nil {
		t.Errorf("Expected nil LowLatencyBuffer")
	}
	err := writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUdpWriter_LineBuffer(t *testing.T) {
	writer, _ := NewUdpWriterWithBuffer("localhost:5000", logger.NewDefaultLogger(), 65536, time.Second)
	if writer.lineBuffer == nil {
		t.Errorf("Expected LineBuffer")
	}
	if writer.lowLatencyBuffer != nil {
		t.Errorf("Expected nil LowLatencyBuffer")
	}
	err := writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUdpWriter_LowLatencyBuffer(t *testing.T) {
	writer, _ := NewUdpWriterWithBuffer("localhost:5000", logger.NewDefaultLogger(), 65537, time.Second)
	if writer.lineBuffer != nil {
		t.Errorf("Expected nil LineBuffer")
	}
	if writer.lowLatencyBuffer == nil {
		t.Errorf("Expected LowLatencyBuffer")
	}
	err := writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
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
	server, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("Could not start UDP server: %v", err)
	}
	defer server.Close()

	// Create a new UDP writer
	writer, err := NewUdpWriter(server.LocalAddr().String(), logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Could not create UDP writer: %v", err)
	}

	// Close the writer
	_ = writer.Close()

	// Write a message
	writer.Write("test message")

	// Check that no message was received
	buffer := make([]byte, 1024)
	_ = server.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
	_, _, err = server.ReadFrom(buffer)
	// ReadFrom will throw error if no message is received after timeout
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

}

func TestUdpWriter_Write(t *testing.T) {
	// Start a local UDP server
	server, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("Could not start UDP server: %v", err)
	}
	defer server.Close()

	// Create a new UDP writer
	writer, err := NewUdpWriter(server.LocalAddr().String(), logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Could not create UDP writer: %v", err)
	}

	// Write a message
	message := "test message"
	writer.Write(message)

	// Read the message from the UDP server
	buffer := make([]byte, len(message))
	_ = server.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
	n, _, err := server.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Could not read from UDP server: %v", err)
	}

	// Check the message
	if string(buffer[:n]) != message {
		t.Errorf("Expected '%s', got '%s'", message, string(buffer[:n]))
	}
}

func TestUdpWriter_LineBuffer_Write(t *testing.T) {
	// Start a local UDP server
	server, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("Could not start UDP server: %v", err)
	}
	defer server.Close()

	// Create a new UDP writer
	writer, err := NewUdpWriterWithBuffer(server.LocalAddr().String(), logger.NewDefaultLogger(), 20, 5*time.Second)
	if err != nil {
		t.Fatalf("Could not create UDP writer: %v", err)
	}

	// Write messages and overflow the buffer, to trigger a flush
	writer.Write("message1")
	writer.Write("message2")
	writer.Write("message3")

	// Read the message from the UDP server
	buffer := make([]byte, 50)
	_ = server.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
	n, _, err := server.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Could not read from UDP server: %v", err)
	}

	// Check the message
	expected := "c:spectator-go.lineBuffer.overflows:1"
	if string(buffer[:n]) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(buffer[:n]))
	}

	// Read the message from the UDP server
	buffer = make([]byte, 50)
	_ = server.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
	n, _, err = server.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("Could not read from UDP server: %v", err)
	}

	// Check the message
	expected = "message1\nmessage2\nmessage3"
	if string(buffer[:n]) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(buffer[:n]))
	}
}

func TestConcurrentWrites(t *testing.T) {
	messagesPerThread := 1000
	writerThreadCount := 4
	var lines []string

	// Start a local UDP server
	server, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("Could not start UDP server: %v", err)
	}
	defer server.Close()

	// Create a new UDP writer
	writer, err := NewUdpWriter(server.LocalAddr().String(), logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Could not create UDP writer: %v", err)
	}
	defer writer.Close()

	var writerWg sync.WaitGroup
	var readerWg sync.WaitGroup

	reader := func() {
		defer readerWg.Done()

		for {
			// read line from UDP server
			buffer := make([]byte, 1024)
			_ = server.SetReadDeadline(time.Now().Add(time.Second)) // prevent infinite blocking
			n, _, err := server.ReadFrom(buffer)
			if err != nil {
				t.Errorf("Error reading from UDP server: %v", err)
				break
			}

			line := string(buffer[:n])

			if line == "done" {
				break
			}

			lines = append(lines, line)
		}
	}

	readerWg.Add(1)
	go reader()

	writerFunc := func(n int) {
		defer writerWg.Done()
		base := n * messagesPerThread
		for i := 0; i < messagesPerThread; i++ {
			writer.Write(strconv.Itoa(base + i))
		}
	}

	// Start writer goroutines
	for j := 0; j < writerThreadCount; j++ {
		writerWg.Add(1)
		go writerFunc(j)
	}

	// Wait writer goroutines to finish
	writerWg.Wait()

	writer.Write("done")

	// Wait for reader goroutine to finish
	readerWg.Wait()

	m := writerThreadCount * messagesPerThread
	if len(lines) != m {
		t.Errorf("Expected %d, got %d", m, len(lines))
	}

	// Create array of expected lines, sort lines and compare both
	expected := make([]int, m)
	for i := 0; i < m; i++ {
		expected[i] = i
	}

	// Convert lines to integers and sort
	intLines := make([]int, len(lines))
	for i, line := range lines {
		value, err := strconv.Atoi(line)
		if err != nil {
			t.Errorf("Error converting line to integer: %v", err)
			return
		}
		intLines[i] = value
	}

	// sort intLines
	sort.Ints(intLines)

	// Compare lines with expected
	for i := 0; i < m; i++ {
		if intLines[i] != expected[i] {
			t.Errorf("Expected %d, got %d", expected[i], intLines[i])
		}
	}
}
