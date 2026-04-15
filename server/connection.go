package server

import (
	"context"
	"log"
	"net"
	"sync"
)

// The connection struct.
type Connection struct {
	conn net.Conn

	// Each channel will be responsible for
	// sending or reading bytes from the connection.
	receiveChan chan []byte
	sendChan    chan []byte

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
	parentCtx context.Context,
	conn net.Conn,
) *Connection {
	ctx, cancel := context.WithCancel(parentCtx)

	return &Connection{
		conn:        conn,
		receiveChan: make(chan []byte, 1024),
		sendChan:    make(chan []byte, 1024),
		ctx:         ctx,
		cancel:      cancel,
		closeOnce:   sync.Once{},
		status:      true,
	}
}

func Monitor(c *Connection, serverReceive chan []byte, serverDeleteConn chan string) {
	go monitorReceive(c, serverReceive, serverDeleteConn)
	go monitorSend(c)
}

// This function will be responsible for checking
// for possible data recievals from the connection.
// It will be spawned up as a goroutine to run in
// the background.
func monitorReceive(c *Connection, serverReceive chan []byte, serverDeleteConn chan string) {
loop:
	for {
		// Creating the buffer to store the data read.
		buf := make([]byte, 1024)
		// Since the conn.Read() operation is a blocking one,
		// the execution will be stuck here until something is
		// received.
		n, err := c.conn.Read(buf)
		// Once the client closes the connection, the conn.Read()
		// will return an io.EOF error.
		if err != nil {
			select {
			// Signalling the server that the connection has been
			// closed.
			case serverDeleteConn <- c.conn.RemoteAddr().String():
			case <-c.ctx.Done():
				//Closing the connection
				closeConn(c)
			}
			break loop // Ending the function.
		}
		// Once the data is read, we must crop it to the actual
		// size of the message.
		msg := buf[:n]
		log.Printf("message received from %s: %s", c.conn.RemoteAddr().String(), msg)

		// Sending the message to the server
		serverReceive <- msg
		log.Printf("message sent to serverReceive")
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
			log.Printf("received context.Done(). Ending connection.")
			closeConn(c)
			break loop // Breaking the loop and ending the function.
		case buf = <-c.sendChan:
			log.Printf("received sendChan (%s): %s\n", c.conn.RemoteAddr().String(), string(buf))
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
		log.Printf("connection with %s was closed\n", c.conn.RemoteAddr().String())
	})
}
