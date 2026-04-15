package server

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
)

// The connection struct.
type Connection struct {
	conn net.Conn

	// Each channel will be responsible for
	// sending or reading bytes from the connection.
	serverReceive chan []byte
	serverSend    chan []byte

	// Each connection will have a son context.
	ctx    context.Context
	cancel context.CancelFunc

	// This field is responsible for making sure
	// that all the fields are closed just once.
	// Let´s say the client and the server try to
	// close the connection at the same time. In this
	// case, we would get a panic.
	closeOnce sync.Once

	// This field will indicate if the Connection
	// is active or closed.
	status bool
}

// Connection constructor
func NewConnection(
	ctx context.Context,
	cancel context.CancelFunc,
	conn net.Conn,
	serverReceive chan []byte,
	serverSend chan []byte,
) *Connection {
	return &Connection{
		conn:          conn,
		serverReceive: serverReceive,
		serverSend:    serverSend,
		ctx:           ctx,
		cancel:        cancel,
		closeOnce:     sync.Once{},
		status:        true,
	}
}

func Monitor(c *Connection) {
	go monitorReceive(c)
	go monitorSend(c)
}

// This function will be responsible for checking
// for possible data recievals from the connection.
// It will be spawned up as a goroutine to run in
// the background.
func monitorReceive(c *Connection) {
loop:
	for {
		// Creating the buffer to store the data read.
		buf := make([]byte, 1024)
		// Since the conn.Read() operation is a blocking one,
		// the execution will be stuck here until something is
		// received.
		_, err := c.conn.Read(buf)
		// Once the client closes the connection, the conn.Read()
		// will return an io.EOF error.
		if errors.Is(err, io.EOF) {
			closeConn(c)
			break loop // Ending the function.
		}
		// Once the data is read, we must crop it to the actual
		// size of the message.
		n := len(buf)
		msg := buf[:n]
		// After the receival, we send it to the serverReceive chan.
		c.serverReceive <- msg
	}
}

// Same logic as the receive one. This function will be
// responsible for sending data to the connection. It
// will be listening to the c.serverSend channel waiting
// for new data to arrive.
func monitorSend(c *Connection) {
loop:
	for {
		var buf []byte
		// If the connection was closed by the client,
		// we should not be able to send nothing through the
		// channel.
		select {
		case <-c.ctx.Done():
			closeConn(c)
			break loop // Breaking the loop and ending the function.
		case buf = <-c.serverSend:
		}
		// Once the data is read, we send it to the
		// connection.
		c.conn.Write(buf)
	}
}

// Function to close all fields of the connection.
func closeConn(c *Connection) {
	// All the code inside the Do() function
	// will only be executed once.
	c.closeOnce.Do(func() {
		c.conn.Close()
		c.cancel()
		c.status = false
		close(c.serverReceive)
		close(c.serverSend)
	})
}
