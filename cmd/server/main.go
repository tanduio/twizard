package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
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
	ctx, cancel := context.WithCancel(context.Background())

	srv, err := server.New(address)
	if err != nil {
		log.Fatal("server creating error: ", err)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		if err := srv.Serve(ctx); err != nil {
			log.Println("server was closed with error: ", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan
	}()

	wg.Wait()

	if err := srv.Shutdown(); err != nil {
		log.Println("server shutdown error: ", err)
	}
}
