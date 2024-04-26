package writer

import (
	"github.com/Netflix/spectator-go/spectator/logger"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
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

func TestFileWriterWithUnixgramServer(t *testing.T) {
	t.Skip("os.OpenFile is not suitable for opening a Unix Datagram socket.")

	// Create a channel for messages
	msgCh := make(chan string)

	// Create a new unixgram server
	socketFile := "/tmp/spectatord-test-" + strconv.Itoa(rand.Intn(10000))
	server, err := newUnixgramServer(socketFile, msgCh)
	if err != nil {
		t.Fatalf("Failed to create unixgram server: %v", err)
	}
	defer server.Close()

	// Create a file_writer instance
	writer, err := NewFileWriter(socketFile, logger.NewDefaultLogger()) // Assuming NewFileWriter is a function that creates a new file_writer instance
	if err != nil {
		t.Fatalf("Failed to create writer. Err: %v", err)
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
}
