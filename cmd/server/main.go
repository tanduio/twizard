package main

import (
	"log"
	"net"
	"os"

	flag "github.com/spf13/pflag"
)

var (
	server string
)

func init() {
	flag.StringVarP(&server, "server", "s", "", "sets the listen address for the incoming traffic")

	flag.Parse()
}

func main() {
	l, err := net.Listen("tcp", server)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	srvl := log.New(os.Stdout, "server ", 0)

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		srvl.Printf("New VPN client: %v", conn.RemoteAddr())

		go handleConnection(conn, srvl)
	}
}

func handleConnection(conn net.Conn, logger *log.Logger) {
	defer conn.Close()

	buffer := make([]byte, 1500)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			logger.Printf("Client %v disconnected: %v", conn.RemoteAddr(), err)
			return
		}

		logger.Printf("From %v: %d bytes", conn.RemoteAddr(), n)

		packet := buffer[:n]

		_, err = conn.Write(packet)
		if err != nil {
			logger.Printf("Error echoing to client: %v", err)
			return
		}

		logger.Printf("Echoed %d bytes back", n)
	}
}
