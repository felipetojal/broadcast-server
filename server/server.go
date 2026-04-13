package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"sync"
	"syscall"

	"github.com/felipetojal/broadcast-server/logger"
)

type Server struct {
	addr   string
	port   string
	logger *slog.Logger
	conns  map[net.Conn]struct{}
	mu     sync.RWMutex
}

func NewServer(addr, port string, logger *slog.Logger) *Server {
	return &Server{
		addr:   addr,
		port:   port,
		logger: logger,
		conns:  make(map[net.Conn]struct{}),
	}
}

// Public function to start the server.
func Start(s *Server) error {
	l, err := createListener(s)
	if err != nil {
		return err
	}

	listen(l, s)

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

		conn.Write([]byte("hello\n"))
		logger.LogInfo(s.logger, "connection established: "+conn.RemoteAddr().String())

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
	for k, _ := range s.conns {
		fmt.Printf("Remote conn addr: %v\n", k.RemoteAddr())
		k.Close()
	}
	logger.LogInfo(s.logger, "all conns were closed.")
}
