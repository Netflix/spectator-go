package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

// newUnixgramServer creates a new unixgram server and returns a connection to it.
// The server listens for incoming messages and sends them to the provided channel.
func newUnixgramServer(socketFile string, msgCh chan string) (*net.UnixConn, error) {
	if err := os.RemoveAll(socketFile); err != nil {
		return nil, err
	}

	addr := &net.UnixAddr{
		Name: socketFile,
		Net:  "unixgram",
	}

	conn, err := net.ListenUnixgram("unixgram", addr)
	if err != nil {
		return nil, err
	}

	go handleConnections(conn, msgCh)

	return conn, nil
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

func TestUnixgramWriter_Integration(t *testing.T) {
	// Create a channel for messages
	msgCh := make(chan string)

	// Create a new unixgram server
	socketFile := "/tmp/test_spectator_unixgram.sock"
	server, err := newUnixgramServer(socketFile, msgCh)
	if err != nil {
		t.Fatalf("Failed to create unixgram server: %v", err)
	}
	defer server.Close()

	// Create a file_writer instance
	writer, err := NewUnixgramWriter(socketFile, logger.NewDefaultLogger())
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Write some messages
	messages := []string{"message1", "message2", "message3"}
	for _, msg := range messages {
		writer.Write(msg)
	}

	// Read the messages from the unixgram server
	for _, originalMsg := range messages {
		select {
		case receivedMsg := <-msgCh:
			if receivedMsg != originalMsg {
				t.Errorf("Received message '%s' does not match original message '%s'", receivedMsg, originalMsg)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for message")
		}
	}

	// allow some time for message logging to complete
	time.Sleep(2 * time.Millisecond)
}

func TestUnixgramWriterWithBuffer_Integration(t *testing.T) {
	// Create a channel for messages
	msgCh := make(chan string)

	// Create a new unixgram server
	socketFile := "/tmp/test_spectator_unixgram_buffer.sock"
	server, err := newUnixgramServer(socketFile, msgCh)
	if err != nil {
		t.Fatalf("Failed to create unixgram server: %v", err)
	}
	defer server.Close()

	// Create a file_writer instance
	writer, err := NewUnixgramWriterWithBuffer(socketFile, logger.NewDefaultLogger(), 20)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Write some messages
	messages := []string{"message1", "message2", "message3"}
	for _, msg := range messages {
		writer.Write(msg)
	}

	// Read the messages from the unixgram server
	expected := "message1\nmessage2\nmessage3"
	select {
	case receivedMsg := <-msgCh:
		if receivedMsg != expected {
			t.Errorf("Received message '%s' does not match original message '%s'", receivedMsg, expected)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}

	// allow some time for message logging to complete
	time.Sleep(2 * time.Millisecond)
}
