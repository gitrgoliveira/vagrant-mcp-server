package server

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/vagrant-mcp/server/internal/exec"
	"github.com/vagrant-mcp/server/internal/resources"
	"github.com/vagrant-mcp/server/internal/sync"
	"github.com/vagrant-mcp/server/internal/tools"
	"github.com/vagrant-mcp/server/internal/vm"
	"github.com/vagrant-mcp/server/pkg/mcp"
)

// Server manages the MCP server instance
type Server struct {
	mcpServer  *mcp.Server
	vmManager  *vm.Manager
	syncEngine *sync.Engine
	executor   *exec.Executor
	doneCh     chan struct{}
}

// NewServer creates a new MCP server for Vagrant
func NewServer() (*Server, error) {
	// Initialize the MCP server
	mcpServer, err := mcp.NewServer("vagrant-mcp-server", "Vagrant Development VM MCP Server")
	if err != nil {
		return nil, err
	}

	// Initialize VM manager
	vmManager, err := vm.NewManager()
	if err != nil {
		return nil, err
	}

	// Initialize sync engine
	syncEngine, err := sync.NewEngine()
	if err != nil {
		return nil, err
	}

	// Initialize executor
	executor, err := exec.NewExecutor(vmManager, syncEngine)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		mcpServer:  mcpServer,
		vmManager:  vmManager,
		syncEngine: syncEngine,
		executor:   executor,
		doneCh:     make(chan struct{}),
	}

	// Register all resources and tools
	if err := srv.registerComponents(); err != nil {
		return nil, err
	}

	return srv, nil
}

// registerComponents registers all MCP resources and tools
func (s *Server) registerComponents() error {
	// Register tools
	if err := tools.RegisterVMTools(s.mcpServer, s.vmManager, s.syncEngine); err != nil {
		return err
	}

	if err := tools.RegisterExecTools(s.mcpServer, s.vmManager, s.syncEngine, s.executor); err != nil {
		return err
	}

	if err := tools.RegisterEnvTools(s.mcpServer, s.vmManager, s.executor); err != nil {
		return err
	}

	if err := tools.RegisterSyncTools(s.mcpServer, s.vmManager, s.syncEngine); err != nil {
		return err
	}

	// Register resources
	if err := resources.RegisterVMResources(s.mcpServer, s.vmManager); err != nil {
		return err
	}

	if err := resources.RegisterLogResources(s.mcpServer); err != nil {
		return err
	}

	if err := resources.RegisterNetworkResources(s.mcpServer, s.vmManager); err != nil {
		return err
	}

	if err := resources.RegisterMonitoringResources(s.mcpServer, s.vmManager, s.executor); err != nil {
		return err
	}

	// Register file resources
	if err := resources.RegisterFileResources(s.mcpServer, s.vmManager, s.executor); err != nil {
		return err
	}

	// Register environment resources
	if err := resources.RegisterEnvironmentResources(s.mcpServer, s.vmManager, s.executor); err != nil {
		return err
	}

	// Register services resources
	if err := resources.RegisterServicesResources(s.mcpServer, s.vmManager, s.executor); err != nil {
		return err
	}

	log.Info().Msg("All MCP components registered")
	return nil
}

// Start begins the MCP server with the given context for cancellation
func (s *Server) Start(ctx context.Context) error {
	// Start the MCP server
	if err := s.mcpServer.Start(); err != nil {
		return err
	}

	// Monitor for context cancellation
	go func() {
		<-ctx.Done()
		log.Info().Msg("Context canceled, stopping server")
		s.Stop()
	}()

	log.Info().Msg("Vagrant MCP Server started successfully")
	return nil
}

// Stop halts the MCP server gracefully
func (s *Server) Stop() error {
	log.Info().Msg("Stopping Vagrant MCP Server...")

	// Stop the MCP server
	if err := s.mcpServer.Stop(); err != nil {
		log.Error().Err(err).Msg("Error stopping MCP server")
	}

	// Clean up resources
	s.syncEngine.Close()
	s.vmManager.Close()

	// Signal completion
	close(s.doneCh)
	return nil
}

// Done returns a channel that's closed when the server has completely shut down
func (s *Server) Done() <-chan struct{} {
	return s.doneCh
}
