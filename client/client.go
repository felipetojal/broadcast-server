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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Creating the client.
	c := newClient(ctx, cancel, conn)

	// Once we connected to the client, we must
	// be able to send and receive messages from
	// the server.
	go runClient(c)
}

func runClient(c *Client) {
	sendServer(c)
	listenServer(c)
}

func sendServer(c *Client) {
	for {

	}
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
		fmt.Println("Received: ", msg)
	}
}

// Function encapsulating the close logic.
func closeClient(c *Client) {
	c.closeOnce.Do(func() {
		c.conn.Close()
	})
}
