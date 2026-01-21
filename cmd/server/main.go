package main

import (
	"log"
	"net"

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

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		log.Printf("New VPN client: %v", conn.RemoteAddr())

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

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
