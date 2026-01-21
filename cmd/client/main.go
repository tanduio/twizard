package main

import (
	"log"
	"net"
	"sync"

	flag "github.com/spf13/pflag"

	"github.com/tanduio/twizard/internal/client"
)

var (
	proxy  string
	device string
)

func init() {
	flag.StringVarP(&proxy, "proxy", "p", "", "defines the target server for traffic forwarding")
	flag.StringVarP(&device, "device", "d", "", "specifies the outgoing interface (e.g., tun0)")

	flag.Parse()
}

func main() {
	tun, err := client.OpenTunInterface(device)
	if err != nil {
		panic(err)
	}
	defer tun.Close()

	wg := sync.WaitGroup{}

	if len(proxy) == 0 {
		log.Fatal("proxy cannot be empty")
	}

	conn, err := net.Dial("tcp", proxy)
	if err != nil {
		log.Fatal("Failed to connect to server:", err)
	}
	defer conn.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()

		client.ForwardTunToTCP(tun, conn)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		client.ForwardTCPToTun(conn, tun)
	}()

	wg.Wait()
}
