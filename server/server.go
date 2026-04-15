package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"syscall"

	"github.com/felipetojal/broadcast-server/logger"
)

// Struct to represent the server.
type Server struct {
	// The address and port associated to the server.
	addr string
	port string

	// The logger responsible for logging messages
	// and possible errors.
	logger *slog.Logger

	// The server has a map of all active connections.
	conns map[string](*Connection)

	// The server has two channels. One channel
	// is responsible for receiving the messages
	// from the clients and the other is responsible
	// for getting these messages and streaming them
	// out to the other clients.
	serverReceive chan []byte
	serverSend    chan []byte
}

// Server constructor
func NewServer(addr, port string, logger *slog.Logger) *Server {
	return &Server{
		addr:          addr,
		port:          port,
		logger:        logger,
		conns:         make(map[string](*Connection)),
		serverReceive: make(chan []byte, 1024),
		serverSend:    make(chan []byte, 1024),
	}
}

// Public function to start the server.
func Start(s *Server) error {
	l, err := createListener(s)
	if err != nil {
		return err
	}
	// Starting the server.
	listen(l, s)
	// If there is no error, return nil.
	return nil
}

// Core function responsible for listenning
// and accepting remote connections.
func listen(l *net.Listener, s *Server) {
	// Works as the shutdown. Once one of the
	// shutdown signals is read from the input,
	// the cancel() function will be executed
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	// Dereferencing the pointer
	lr := *l

	// Goroutine to listen for syscalls.
	go func() {
		// Waiting for the context termination.
		// (syscall)
		<-ctx.Done()
		logger.LogInfo(s.logger, "shutting server down.")

		// Closing the connections.
		closeConns(s)

		// Closing the listener.
		lr.Close()
	}()

	// 	Before the main loop, we need to spawn
	// a function responsible for handling
	// the server channels

	// Core logic of the server. Here it will
	// listen for external connections.
loop:
	for {
		conn, err := lr.Accept()
		if errors.Is(err, net.ErrClosed) {
			logger.LogInfo(s.logger, "closing listener for shutdown.")
			break loop
		} else if err != nil {
			logger.LogError(s.logger, err.Error())
			continue
		}
		logger.LogInfo(s.logger, "connection established: "+conn.RemoteAddr().String())

		// Creating the Connection.
		c := NewConnection(ctx, cancel, conn, s.serverReceive, s.serverSend)

		// Adding the Connection.
		addConn(s, c)

		// Creating the go routine to handle
		// each connection individually.
		go func(con *Connection) {
			Monitor(c)
		}(c)
	}
}

// Function to create the listener responsible
// for handling the connections.
func createListener(s *Server) (*net.Listener, error) {
	// Creating the listener and checking for errors.
	l, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		logger.LogError(s.logger, err.Error())
		return nil, err
	}
	// Logging the successful creation
	logger.LogInfo(s.logger, "listener successfully created")
	// Returning the listener
	return &l, nil
}

// Function to be called once the shutdown is activated.
// It will iterate over the connections and close each one.
func closeConns(s *Server) {
	logger.LogInfo(s.logger, "closing all conns.")
	for k, v := range s.conns {
		fmt.Printf("Remote conn addr: %v\n", k)
		closeConn(v)
	}
	logger.LogInfo(s.logger, "all conns were closed.")
}

// Function to add a new connection to the
// server map.
func addConn(s *Server, c *Connection) {
	// Each connection will be associated to its
	// remote address.
	rAddr := c.conn.RemoteAddr().String()
	s.conns[rAddr] = c
}
