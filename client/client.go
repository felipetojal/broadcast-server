package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// This Client struct will encapsulate the connection
// to the server.
type Client struct {
	// The connection established.
	conn net.Conn

	// The context associated to the client.
	ctx context.Context

	// The cancel function associated to the client.
	cancel context.CancelFunc

	// The scanner will be responsible for reading the
	// messages from the terminal.
	scanner *bufio.Scanner

	// This attribute is responsible for making sure
	// everything is closed only once.
	closeOnce sync.Once
}

// Client constructor.
func newClient(ctx context.Context, cancel context.CancelFunc, conn net.Conn) *Client {
	return &Client{
		conn:      conn,
		ctx:       ctx,
		cancel:    cancel,
		scanner:   bufio.NewScanner(os.Stdin),
		closeOnce: sync.Once{},
	}
}

// This function is responsible for connecting
// to the server.
//
// If the connection fails, it will stop.
func establishConn(serverAddr string) {
	// Connecting to the server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatal("error conencting to server:" + serverAddr)
	}

	// This will receive signals from the terminal
	// indicating exit.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Creating the client.
	c := newClient(ctx, cancel, conn)

	// Once we connected to the client, we must
	// be able to send and receive messages from
	// the server.
	go runClient(c)

	// If the context is canceled, we can finish the
	// function.
	<-ctx.Done()
	closeClient(c)
}

func runClient(c *Client) {
	sendServer(c)
	listenServer(c)
}

// This function will listen to the user
// terminal, waiting for a message to be
// written. Once it is written, it will
// be sent to the server.
func sendServer(c *Client) {
	// This is a blocking operation.
	// It will wait until the user hits
	// enter.
	for c.scanner.Scan() {
		msg := c.scanner.Bytes()

		// Writing the message to the connection.
		_, err := c.conn.Write(msg)
		if err != nil {
			log.Println("error sending message: ", string(msg))
			// If an error happened, we call
			// the context cancel.
			c.cancel()
			return
		}
	}

	if err := c.scanner.Err(); err != nil {
		log.Println("error reading stdin: ", err.Error())
	}
	c.cancel()
}

// This function will run in the background and will listen to the server
func listenServer(c *Client) {
	for {
		// Creating the buffer
		buf := make([]byte, 1024)
		// Receiving the bytes from the server.
		// This is a blocking operation, so the
		// code flow will be stuck here.
		n, err := c.conn.Read(buf)
		if err != nil {
			// If there is an error reading,
			// the connection will be closed.
			c.cancel()
			return
		}

		msg := buf[:n]
		fmt.Println("Received: ", string(msg))
	}
}

// Function encapsulating the close logic.
func closeClient(c *Client) {
	c.closeOnce.Do(func() {
		c.conn.Close()
	})
}
