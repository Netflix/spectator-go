package writer

import (
	"errors"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

const testUnixgramSocket = "/tmp/spectator-go_unixgram.sock"

// newUnixgramServer creates a new unixgram server and returns a connection to it.
// The server listens for incoming messages and sends them to the provided channel.
func newUnixgramServer() (*net.UnixConn, chan string, error) {
	if err := os.RemoveAll(testUnixgramSocket); err != nil {
		return nil, nil, err
	}

	addr := &net.UnixAddr{
		Name: testUnixgramSocket,
		Net:  "unixgram",
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		return nil, nil, err
	}

	messages := make(chan string)
	go handleConnections(conn, messages)

	return conn, messages, nil
}

func handleConnections(conn *net.UnixConn, msgCh chan string) {
	buffer := make([]byte, 1024)

	for {
		n, _, err := conn.ReadFromUnix(buffer)
		if err != nil {
			return
		}
		data := string(buffer[:n])
		// Send received data to channel
		msgCh <- data
		log.Printf("Received message '%s'", data)
	}
}

func readMessage(messages chan string) (string, error) {
	select {
	case message := <-messages:
		return message, nil
	case <-time.After(time.Second):
		return "", errors.New("timeout waiting for message")
	}
}

func TestUnixgramWriter_NoBuffer(t *testing.T) {
	// Create server
	server, _, serverErr := newUnixgramServer()
	if serverErr != nil {
		t.Fatalf("Failed to create unixgram server: %v", serverErr)
	}
	defer server.Close()

	// Create writer
	writer, err := NewUnixgramWriterWithBuffer(testUnixgramSocket, logger.NewDefaultLogger(), 0, time.Second)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	if writer.lineBuffer != nil {
		t.Errorf("Expected nil LineBuffer")
	}
	if writer.lowLatencyBuffer != nil {
		t.Errorf("Expected nil LowLatencyBuffer")
	}
	err = writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUnixgramWriter_LineBuffer(t *testing.T) {
	// Create server
	server, _, serverErr := newUnixgramServer()
	if serverErr != nil {
		t.Fatalf("Failed to create unixgram server: %v", serverErr)
	}
	defer server.Close()

	// Create writer
	writer, err := NewUnixgramWriterWithBuffer(testUnixgramSocket, logger.NewDefaultLogger(), 65536, time.Second)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	if writer.lineBuffer == nil {
		t.Errorf("Expected LineBuffer")
	}
	if writer.lowLatencyBuffer != nil {
		t.Errorf("Expected nil LowLatencyBuffer")
	}
	err = writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUnixgramWriter_LowLatencyBuffer(t *testing.T) {
	// Create server
	server, _, serverErr := newUnixgramServer()
	if serverErr != nil {
		t.Fatalf("Failed to create unixgram server: %v", serverErr)
	}
	defer server.Close()

	// Create writer
	writer, err := NewUnixgramWriterWithBuffer(testUnixgramSocket, logger.NewDefaultLogger(), 65537, time.Second)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	if writer.lineBuffer != nil {
		t.Errorf("Expected nil LineBuffer")
	}
	if writer.lowLatencyBuffer == nil {
		t.Errorf("Expected LowLatencyBuffer")
	}
	err = writer.Close()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUnixgramWriter_Write(t *testing.T) {
	// Create server
	server, msgCh, serverErr := newUnixgramServer()
	if serverErr != nil {
		t.Fatalf("Failed to create unixgram server: %v", serverErr)
	}
	defer server.Close()

	// Create writer
	writer, err := NewUnixgramWriter(testUnixgramSocket, logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Write messages
	messages := []string{"message1", "message2", "message3"}
	for _, msg := range messages {
		writer.Write(msg)
	}

	// Read messages from server
	for _, origMsg := range messages {
		recvMsg, recvErr := readMessage(msgCh)
		if recvErr != nil {
			t.Errorf("Failed to receive message: %v", recvErr)
		}
		if recvMsg != origMsg {
			t.Errorf("Received message '%s' does not match original message '%s'", recvMsg, origMsg)
		}
	}

	// Allow time for messages to deliver
	time.Sleep(2 * time.Millisecond)
}

func TestUnixgramWriter_LineBuffer_Write(t *testing.T) {
	// Create server
	server, msgCh, serverErr := newUnixgramServer()
	if serverErr != nil {
		t.Fatalf("Failed to create unixgram server: %v", serverErr)
	}
	defer server.Close()

	// Create writer
	writer, err := NewUnixgramWriterWithBuffer(testUnixgramSocket, logger.NewDefaultLogger(), 20, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Write messages
	messages := []string{"message1", "message2", "message3"}
	for _, msg := range messages {
		writer.Write(msg)
	}

	// Read messages from server
	expected := "c:spectator-go.lineBuffer.overflows:1"
	recvMsg, recvErr := readMessage(msgCh)
	if recvErr != nil {
		t.Errorf("Failed to receive message: %v", recvErr)
	}
	if recvMsg != expected {
		t.Errorf("Received message '%s' does not match original message '%s'", recvMsg, expected)
	}

	expected = "message1\nmessage2\nmessage3"
	recvMsg, recvErr = readMessage(msgCh)
	if recvErr != nil {
		t.Errorf("Failed to receive message: %v", recvErr)
	}
	if recvMsg != expected {
		t.Errorf("Received message '%s' does not match original message '%s'", recvMsg, expected)
	}

	// Allow time for messages to deliver
	time.Sleep(2 * time.Millisecond)
}
