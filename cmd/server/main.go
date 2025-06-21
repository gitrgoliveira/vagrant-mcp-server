package main

import (
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/handlers"
	"github.com/vagrant-mcp/server/internal/resources"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/utils"
	"github.com/vagrant-mcp/server/internal/vm"
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

	// Check if Vagrant CLI is installed
	if err := utils.CheckVagrantInstalled(); err != nil {
		log.Fatal().Err(err).Msg("Vagrant CLI is required to run this server")
	}
	log.Info().Msg("Vagrant CLI detected")

	// Initialize VM manager, sync engine, and executor
	vmManager, err := vm.NewManager()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create VM manager")
	}

	syncEngine, err := sync.NewEngine()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create sync engine")
	}

	adapterVM := &exec.VMManagerAdapter{Real: vmManager}
	adapterSync := &exec.SyncEngineAdapter{Real: syncEngine}

	executor, err := exec.NewExecutor(adapterVM, adapterSync)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create executor")
	}

	// Create a new MCP server with recovery middleware
	srv := server.NewMCPServer(
		"Vagrant Development VM MCP Server",
		"1.0.0",
		server.WithRecovery(),
	)

	// Register all tools using the MCP-go implementation
	handlers.RegisterVMTools(srv, adapterVM, adapterSync)
	handlers.RegisterExecTools(srv, adapterVM, adapterSync, executor)
	handlers.RegisterEnvTools(srv, adapterVM, executor)
	handlers.RegisterSyncTools(srv, adapterVM, adapterSync)

	// Register resources using the MCP-go implementation
	resources.RegisterMCPResources(srv, adapterVM, executor)

	// Determine which transport to use
	transportType := os.Getenv("MCP_TRANSPORT")
	if transportType == "" {
		transportType = "stdio" // Default to stdio if not specified
	}

	log.Info().Str("transport", transportType).Msg("Vagrant MCP Server starting")

	// Start the server with the selected transport
	switch transportType {
	case "stdio":
		// Start with stdio transport
		log.Info().Msg("Starting with STDIO transport")
		if err := server.ServeStdio(srv); err != nil {
			log.Fatal().Err(err).Msg("STDIO server error")
		}
	case "sse":
		// Start with SSE transport
		port := os.Getenv("MCP_PORT")
		if port == "" {
			port = "8080" // Default port
		}
		log.Info().Str("port", port).Msg("Starting with SSE transport")
		sseServer := server.NewSSEServer(srv)
		if err := sseServer.Start(":" + port); err != nil {
			log.Fatal().Err(err).Msg("SSE server error")
		}
	default:
		log.Fatal().Str("transport", transportType).Msg("Unsupported transport type")
	}

	log.Info().Msg("Vagrant MCP Server shutdown complete")
}
