package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// TransportType defines the transport mechanism for the MCP server
type TransportType string

const (
	// StdioTransport uses standard input/output for communication
	StdioTransport TransportType = "stdio"
	// SSETransport uses Server-Sent Events over HTTP for communication
	SSETransport TransportType = "sse"
)

// Server represents an MCP server instance
type Server struct {
	ID        string
	Name      string
	Version   string
	Transport TransportType
	Port      int
	tools     map[string]Tool
	resources map[string]Resource
	running   bool
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// Tool represents an MCP tool
type Tool interface {
	Name() string
	Description() string
	Execute(params map[string]interface{}) (interface{}, error)
}

// Resource represents an MCP resource
type Resource interface {
	Name() string
	Description() string
	Get(path string) (interface{}, error)
}

// ServerConfig contains initialization parameters for an MCP server
type ServerConfig struct {
	ID        string
	Name      string
	Version   string
	Transport TransportType
	Port      int
}

// NewServer creates a new MCP server with the given configuration
func NewServer(id string, name string) (*Server, error) {
	// Set defaults
	version := "1.0.0"
	transport := StdioTransport
	port := 3000

	// Override with environment variables if present
	if envTransport := os.Getenv("MCP_TRANSPORT"); envTransport != "" {
		transport = TransportType(envTransport)
	}

	if envPort := os.Getenv("MCP_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	return &Server{
		ID:        id,
		Name:      name,
		Version:   version,
		Transport: transport,
		Port:      port,
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}, nil
}

// RegisterTool registers a tool with the MCP server
func (s *Server) RegisterTool(tool Tool) error {
	name := tool.Name()
	if _, exists := s.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}

	s.tools[name] = tool
	return nil
}

// RegisterResource registers a resource with the MCP server
func (s *Server) RegisterResource(resource Resource) error {
	name := resource.Name()
	if _, exists := s.resources[name]; exists {
		return fmt.Errorf("resource %s already registered", name)
	}

	s.resources[name] = resource
	return nil
}

// Start begins the MCP server operations
func (s *Server) Start() error {
	// Initialize capabilities document
	capabilities := map[string]interface{}{
		"id":        s.ID,
		"name":      s.Name,
		"version":   s.Version,
		"tools":     s.getToolsDefinitions(),
		"resources": s.getResourceDefinitions(),
	}

	// Serialize capabilities
	capabilitiesJSON, err := json.MarshalIndent(capabilities, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	// Start the appropriate transport
	switch s.Transport {
	case StdioTransport:
		return s.startStdioTransport(capabilitiesJSON)
	case SSETransport:
		return s.startSSETransport(capabilitiesJSON)
	default:
		return fmt.Errorf("unsupported transport type: %s", s.Transport)
	}
}

// startStdioTransport initializes the stdio transport
func (s *Server) startStdioTransport(capabilitiesJSON []byte) error {
	s.running = true

	// Send capabilities document as the first message
	if err := s.writeJSONRPC(os.Stdout, "capabilities", map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]interface{}{
			"capabilities": json.RawMessage(capabilitiesJSON),
		},
	}); err != nil {
		return fmt.Errorf("failed to write capabilities: %w", err)
	}

	// Start processing in a goroutine
	go s.processStdio()

	return nil
}

// processStdio handles stdin/stdout communication for the MCP server
func (s *Server) processStdio() {
	defer func() {
		s.running = false
		close(s.doneCh)
	}()

	// Create scanner for stdin
	scanner := bufio.NewScanner(os.Stdin)

	// Use a buffer for larger messages
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// Process incoming messages
	for scanner.Scan() {
		select {
		case <-s.stopCh:
			return
		default:
			// Process the message
			if err := s.handleMessage(scanner.Bytes()); err != nil {
				if err2 := s.writeError(os.Stdout, 0, err); err2 != nil {
					fmt.Fprintf(os.Stderr, "Failed to write error: %v\n", err2)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if err2 := s.writeError(os.Stdout, 0, fmt.Errorf("stdin scanning error: %w", err)); err2 != nil {
			fmt.Fprintf(os.Stderr, "Failed to write error: %v\n", err2)
		}
	}
}

// handleMessage processes an incoming JSON-RPC message
func (s *Server) handleMessage(data []byte) error {
	// Parse the message
	var message struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id,omitempty"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("invalid JSON-RPC message: %w", err)
	}

	// Verify JSON-RPC version
	if message.JSONRPC != "2.0" {
		return fmt.Errorf("unsupported JSON-RPC version: %s", message.JSONRPC)
	}

	// Handle message based on method
	switch message.Method {
	case "tool:execute":
		return s.handleToolExecute(os.Stdout, message.ID, message.Params)
	case "resource:get":
		return s.handleResourceGet(os.Stdout, message.ID, message.Params)
	case "shutdown":
		return s.handleShutdown(os.Stdout, message.ID)
	default:
		return fmt.Errorf("unsupported method: %s", message.Method)
	}
}

// handleToolExecute processes a tool execution request
func (s *Server) handleToolExecute(w io.Writer, id interface{}, params json.RawMessage) error {
	var request struct {
		Name   string                 `json:"name"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(params, &request); err != nil {
		return s.writeError(w, id, fmt.Errorf("invalid tool execution parameters: %w", err))
	}

	tool, exists := s.tools[request.Name]
	if !exists {
		return s.writeError(w, id, fmt.Errorf("tool not found: %s", request.Name))
	}

	// Execute the tool
	result, err := tool.Execute(request.Params)
	if err != nil {
		return s.writeError(w, id, fmt.Errorf("tool execution failed: %w", err))
	}

	// Send response
	return s.writeJSONRPC(w, id, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

// handleResourceGet processes a resource get request
func (s *Server) handleResourceGet(w io.Writer, id interface{}, params json.RawMessage) error {
	var request struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(params, &request); err != nil {
		return s.writeError(w, id, fmt.Errorf("invalid resource parameters: %w", err))
	}

	// Parse the URI to get the resource name and path
	parts := strings.SplitN(request.URI, "://", 2)
	if len(parts) != 2 {
		return s.writeError(w, id, fmt.Errorf("invalid resource URI format: %s", request.URI))
	}

	resourceName, path := parts[0], parts[1]
	resource, exists := s.resources[resourceName+"://"]
	if !exists {
		return s.writeError(w, id, fmt.Errorf("resource not found: %s", resourceName))
	}

	// Get the resource
	result, err := resource.Get(path)
	if err != nil {
		return s.writeError(w, id, fmt.Errorf("resource retrieval failed: %w", err))
	}

	// Send response
	return s.writeJSONRPC(w, id, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

// handleShutdown processes a shutdown request
func (s *Server) handleShutdown(w io.Writer, id interface{}) error {
	// Send response before shutting down
	if err := s.writeJSONRPC(w, id, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  true,
	}); err != nil {
		return err
	}

	// Trigger shutdown
	close(s.stopCh)
	return nil
}

// writeError sends a JSON-RPC error response
func (s *Server) writeError(w io.Writer, id interface{}, err error) error {
	return s.writeJSONRPC(w, id, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    -32000,
			"message": err.Error(),
		},
	})
}

// writeJSONRPC writes a JSON-RPC message to the writer
func (s *Server) writeJSONRPC(w io.Writer, id interface{}, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON-RPC message: %w", err)
	}

	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		return fmt.Errorf("failed to write JSON-RPC message: %w", err)
	}

	return nil
}

// startSSETransport initializes the SSE transport
func (s *Server) startSSETransport(capabilitiesJSON []byte) error {
	s.running = true

	// Start HTTP server for SSE transport
	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Send initial capabilities event
		if _, err := fmt.Fprintf(w, "event: initialize\ndata: %s\n\n", capabilitiesJSON); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write SSE event: %v\n", err)
		}

		// Flush the response to ensure the client receives it immediately
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Handle incoming JSON-RPC messages via POST
		if r.Method == "POST" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusBadRequest)
				return
			}

			if err := s.handleMessage(body); err != nil {
				if err2 := s.writeError(w, 0, err); err2 != nil {
					fmt.Fprintf(os.Stderr, "Failed to write SSE error: %v\n", err2)
				}
			}
		}

		// Keep the connection alive until server is stopped
		<-s.stopCh
	})

	// Start HTTP server
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", s.Port),
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}

		s.running = false
		close(s.doneCh)
	}()

	return nil
}

// Stop halts the MCP server operations
func (s *Server) Stop() error {
	if !s.running {
		return nil // Already stopped
	}

	// Signal all goroutines to stop
	close(s.stopCh)

	// Wait for server to completely shut down
	<-s.doneCh

	return nil
}

// Done returns a channel that's closed when the server is completely stopped
func (s *Server) Done() <-chan struct{} {
	return s.doneCh
}

// getToolsDefinitions returns the tool definitions for the capabilities document
func (s *Server) getToolsDefinitions() []map[string]interface{} {
	tools := make([]map[string]interface{}, 0, len(s.tools))

	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
		})
	}

	return tools
}

// getResourceDefinitions returns the resource definitions for the capabilities document
func (s *Server) getResourceDefinitions() []map[string]interface{} {
	resources := make([]map[string]interface{}, 0, len(s.resources))

	for _, resource := range s.resources {
		resources = append(resources, map[string]interface{}{
			"name":        resource.Name(),
			"description": resource.Description(),
		})
	}

	return resources
}
