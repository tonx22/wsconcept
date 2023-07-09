package main

import (
	"github.com/Netflix/go-env"
	"github.com/tonx22/wsconcept/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type environment struct {
	HTTPPort int `env:"HTTP_PORT"`
}

func main() {
	var e environment
	_, err := env.UnmarshalFromEnviron(&e)
	if err != nil {
		log.Fatalf("Can't get environment variables: %v", err)
	}

	err = server.StartNewHTTPServer(e.HTTPPort)
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	<-sigChan
}
