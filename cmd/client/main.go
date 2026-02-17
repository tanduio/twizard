package main

import (
	"flag"
	"log"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/tanduio/twizard/internal/client"
	"github.com/tanduio/twizard/internal/tnet"
)

var (
	proxy             string
	tunInterface      string
	outboundInterface string
)

func init() {
	flag.StringVar(&proxy, "proxy", "", "server address (e.g., vpn.example.com:1194)")
	flag.StringVar(&tunInterface, "tun", "", "TUN interface to read traffic from (e.g., tun0)")
	flag.StringVar(&outboundInterface, "outbound-iface", "", "Network interface for connecting to server (e.g., eth0, wlan0)")

	flag.Parse()
}

func main() {
	tun, err := tnet.OpenRawInterface(tunInterface)
	if err != nil {
		panic(err)
	}
	defer tun.Close()

	wg := sync.WaitGroup{}

	if len(proxy) == 0 {
		log.Fatal("proxy cannot be empty")
	}

	log.Println("Connecting to the server")

	dialer := net.Dialer{
		Timeout: time.Second * 5,
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, outboundInterface)
			})
		},
	}

	log.Println("Connection established")

	conn, err := dialer.Dial("tcp", proxy)
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
