package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tanduio/twizard/internal/server"
	"github.com/tanduio/twizard/internal/tnet"
)

var (
	listen string
	tun    string
)

func init() {
	flag.StringVar(&listen, "listen", "", "local address to listen for connections (e.g., 0.0.0.0:1194)")
	flag.StringVar(&tun, "tun", "", "TUN interface for routing decapsulated traffic (e.g., tun0)")

	flag.Parse()
}

func main() {
	ctx := context.Background()

	tun, err := tnet.OpenTunInterface(tun)
	if err != nil {
		panic(err)
	}
	defer tun.Close()

	srv := server.New(listen, tun)

	go func() {
		if err := srv.ListenAndServe(ctx); err != nil {
			log.Println("server was closed with error: ", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	if err := srv.Shutdown(); err != nil {
		log.Println("server shutdown error: ", err)
	}
}
