package server

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	TimeoutError = errors.New("timeout")
)

type Server struct {
	l            net.Listener
	activeConn   map[net.Conn]struct{}
	activeConnWG sync.WaitGroup

	address string

	mu         sync.Mutex
	inShutdown atomic.Bool
}

func New(address string) *Server {
	return &Server{
		address: address,
	}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.l = l
	s.activeConn = make(map[net.Conn]struct{})

	return s.serve(ctx)
}

func (s *Server) serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("server context canceled")
			return ctx.Err()
		default:
			conn, err := s.l.Accept()
			if err != nil {
				return err
			}

			if s.shuttingDown() {
				return nil
			}

			log.Printf("new client: %v", conn.RemoteAddr())

			go s.handleConnection(conn)
		}
	}
}

func (s *Server) addConn(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeConnWG.Add(1)
	s.activeConn[conn] = struct{}{}
}

func (s *Server) removeConn(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeConnWG.Done()
	delete(s.activeConn, conn)
}

func (s *Server) Shutdown() error {
	s.inShutdown.Store(true)

	s.mu.Lock()
	err := s.l.Close()
	if err != nil {
		log.Println("[ERROR] failed to close net.Listener: " + err.Error())
	}
	s.mu.Unlock()

	gracefullyClosed := make(chan struct{})
	go func() {
		s.activeConnWG.Wait()
		gracefullyClosed <- struct{}{}
	}()

	select {
	case <-gracefullyClosed: // TODO: complex shutdown based on conn state
		return err
	case <-time.After(10 * time.Second):
		return TimeoutError
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.Load()
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.addConn(conn)
	defer s.removeConn(conn)

	buffer := make([]byte, 1500)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Client %v disconnected: %v", conn.RemoteAddr(), err)
			return
		}

		log.Printf("From %v: %d bytes", conn.RemoteAddr(), n)

		packet := buffer[:n]

		_, err = conn.Write(packet)
		if err != nil {
			log.Printf("Error echoing to client: %v", err)
			return
		}

		log.Printf("Echoed %d bytes back", n)
	}
}
