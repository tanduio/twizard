package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"
	"github.com/tanduio/twizard/internal/server"
)

var (
	address string
)

func init() {
	flag.StringVarP(&address, "address", "a", "", "sets the listen address for the incoming traffic")

	flag.Parse()
}

func main() {
	ctx := context.Background()

	srv := server.New(address)

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
