package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"bean/cmd/pestcontrol/internal/server"
	"bean/cmd/pestcontrol/internal/telemetry"
	pserver2 "bean/pkg/pserver/v2"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()

	if err := run(*portNumber); err != nil {
		log.Fatal(err)
	}
}

func run(port int) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shutdown, err := telemetry.SetupOtelSDK(ctx)
	if err != nil {
		return fmt.Errorf("setup otel sdk: %w", err)
	}
	defer shutdown(ctx)

	s := server.New()
	go s.Initialize()

	handler := pserver2.WithMiddleware(
		s.HandleConnection,
		pserver2.LoggingMiddleware,
	)

	return pserver2.ListenServe(ctx, handler, port)
}
