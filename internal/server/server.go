package server

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tanduio/twizard/internal/tnet"
)

var (
	TimeoutError = errors.New("timeout")
)

type Server struct {
	l            net.Listener
	activeConn   map[net.Conn]struct{}
	activeConnWG sync.WaitGroup

	address string
	tun     *tnet.Tun

	mu         sync.Mutex
	inShutdown atomic.Bool
}

func New(address string, tun *tnet.Tun) *Server {
	return &Server{
		address: address,
		tun:     tun,
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
	go s.tun.StartReader() // TODO graceful shutdown

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

			go s.handleConnection(ctx, conn)
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

func (s *Server) handleConnection(ctx context.Context, clientConn net.Conn) {
	defer clientConn.Close()

	buffer := make([]byte, 4000)

	for {
		fmt.Println("\n=== Reading Request ===")
		n, err := clientConn.Read(buffer)
		if err != nil {
			log.Println("Failed to read from client:", err)
			break
		}

		ipPacket := buffer[:n]

		// Update source IP
		copy(ipPacket[12:16], net.ParseIP("192.168.69.1").To4())

		// Recalculate IP checksum
		ipChecksum := tnet.CalculateIPChecksum(ipPacket[:20])
		binary.BigEndian.PutUint16(ipPacket[10:12], ipChecksum)

		// Recalculate TCP checksum
		srcIP := ipPacket[12:16]
		dstIP := ipPacket[16:20]
		tcpData := ipPacket[20:]
		tcpChecksum := tnet.CalculateTCPChecksum(tcpData, srcIP, dstIP)
		binary.BigEndian.PutUint16(ipPacket[36:38], tcpChecksum)

		src := fmt.Sprintf("%d.%d.%d.%d", ipPacket[12], ipPacket[13], ipPacket[14], ipPacket[15])
		dst := fmt.Sprintf("%d.%d.%d.%d", ipPacket[16], ipPacket[17], ipPacket[18], ipPacket[19])
		proto := ipPacket[9]

		log.Printf("[->] TUN->TCP: %s -> %s (proto:%d, %d bytes): version[%d]", src, dst, proto, n, ipPacket[0]>>4)

		ctx, _ := context.WithTimeout(ctx, time.Second*4)

		respCh, err := s.tun.Send(ctx, ipPacket)
		if err != nil {
			log.Printf("Error writing to TUN: %v", err)
		}

		select {
		case <-ctx.Done():
			log.Printf("ctx error: %s\n", ctx.Err().Error())
		case resp := <-respCh:
			n, err = clientConn.Write(resp)
			if err != nil {
				log.Println("Failed to write to client back from destination:", err)
				return
			}
		}
	}
	log.Println("Connection closed")
}