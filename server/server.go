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

// Struct to represent the server.
type Server struct {
	// The address and port associated to the server.
	addr string
	port string

	// The logger responsible for logging messages
	// and possible errors.
	logger *slog.Logger

	// The server has a map of all active connections.
	// The key will be the remote address and the value
	// will be a pointer to the connection.
	conns map[string](*Connection)

	// The server has two channels. One channel
	// is responsible for receiving the messages
	// from the clients and the other is responsible
	// for getting these messages and streaming them
	// out to the other clients.
	serverReceive chan []byte
	serverSend    chan []byte

	// Once the connection signals the close,
	// it will send message to the server containing
	// its remote address. This will allow the server
	// to exclude the connection from the map.
	serverDeleteConn chan string

	// Mutex to control the access to the map,
	// since it is not thread safe.
	mu sync.RWMutex

	// Field responsible for making sure that the channels
	// are closed only once.
	closeOnce sync.Once
}

// Server constructor
func NewServer(addr, port string, logger *slog.Logger) *Server {
	return &Server{
		addr:             addr,
		port:             port,
		logger:           logger,
		conns:            make(map[string](*Connection)),
		serverReceive:    make(chan []byte, 1024),
		serverSend:       make(chan []byte, 1024),
		serverDeleteConn: make(chan string, 100),
		mu:               sync.RWMutex{},
		closeOnce:        sync.Once{},
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

		// Closing the server channels.
		closeServerChan(s)
	}()

	// 	Before the main loop, we need to spawn
	// a function responsible for handling
	// the server channels.
	go propagateMessage(ctx, s)
	go checkClosedConn(ctx, s)

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
		c := NewConnection(ctx, conn)

		// Adding the Connection.
		addConn(s, c)

		// Creating the go routine to handle
		// each connection individually.
		go func(con *Connection) {
			Monitor(c, s.serverReceive, s.serverDeleteConn)
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
	// Since we are accessing the map, we must lock the mutex.
	s.mu.Lock()
	defer s.mu.Unlock()

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
	// Since the map is being accessed by multiple
	// goroutines and it is not thread-safe, we net
	// a mutex to control its access.
	s.mu.Lock()
	defer s.mu.Unlock()
	// Each connection will be associated to its
	// remote address.
	rAddr := c.conn.RemoteAddr().String()
	s.conns[rAddr] = c
}

// This function will be responsible for listening to
// the s.serverReceive channel and publish the message
// to the s.serverSend channel.
func propagateMessage(ctx context.Context, s *Server) {
	for {
		select {
		// If we read a cancellation signal from the
		// context, we end the function execution.
		case <-ctx.Done():
			closeServerChan(s)
			return
		// If a message is read from the s.serverReceive,
		// we propagate it to the connections.
		case msg := <-s.serverReceive:
			sendMessage(s, msg)
		}
	}
}

// This function will propagate the message received
// by the server to all connections in the map.
func sendMessage(s *Server, msg []byte) {
	// Controlling the mutex. We lock it only for
	// read since we are no directly altering the map,
	// just iterating over its elements.
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Iterating over the map.
	for _, v := range s.conns {
		if v.status {
			v.sendChan <- msg
		}
	}
}

// This function will listen for the channel dedicated
// to closing connections. It will help free up memory space.
func checkClosedConn(ctx context.Context, s *Server) {
	for {
		var rAddr string
		// Blocking the code and waiting for the signal
		select {
		case <-ctx.Done():
			return
		case rAddr = <-s.serverDeleteConn:
			logger.LogInfo(s.logger, "delete connection from map:"+rAddr)
		}

		// Since we are manipulating the connections map,
		// we must lock it.
		s.mu.Lock()
		defer s.mu.Unlock()

		// Deleting from the map.
		delete(s.conns, rAddr)
	}
}

// Function to guarantee that the channels
// will be closed only one time.
func closeServerChan(s *Server) {
	s.closeOnce.Do(func() {
		close(s.serverReceive)
		close(s.serverSend)
		close(s.serverDeleteConn)
	})
}
