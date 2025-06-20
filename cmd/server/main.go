package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/server"
)

func main() {
	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Set log level from environment or default to info
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log.Info().Msg("Starting Vagrant MCP Server")

	// Create server with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize the server
	srv, err := server.NewServer()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}

	// Start the server
	if err := srv.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Info().Msgf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Wait for server to complete
	<-srv.Done()
	log.Info().Msg("Vagrant MCP Server shutdown complete")
}
