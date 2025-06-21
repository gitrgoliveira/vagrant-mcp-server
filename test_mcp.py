#!/usr/bin/env python3
import json
import subprocess
import sys

def send_request(method, params=None):
    """Send a JSON-RPC request to the MCP server over stdio."""
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method
    }
    if params:
        request["params"] = params
    
    return json.dumps(request)

def main():
    if len(sys.argv) < 2:
        print("Usage: test_mcp.py <method> [params_json] [--execute]")
        print("\nExamples:")
        print("  test_mcp.py initialize")
        print("  test_mcp.py tools/list")
        print("  test_mcp.py tools/call '{\"name\":\"create_dev_vm\",\"arguments\":{\"name\":\"test-vm\",\"project_path\":\"/path/to/project\"}}'")
        print("  test_mcp.py resources/read '{\"uri\":\"devvm://status\"}'")
        print("  test_mcp.py initialize '{\"capabilities\": {\"resource_capabilities\": {\"subscribe\": true}}}' --execute")
        sys.exit(1)
    
    method = sys.argv[1]
    params = None
    execute = False
    
    # Parse arguments
    for arg in sys.argv[2:]:
        if arg == "--execute":
            execute = True
        elif arg.startswith("{"):
            try:
                params = json.loads(arg)
            except json.JSONDecodeError:
                print(f"Error: Invalid JSON: {arg}")
                sys.exit(1)
    
    # Print the request that would be sent
    request_json = send_request(method, params)
    print(f"Request that would be sent to MCP server:")
    print(request_json)
    print("\nTo test with a running server, use:")
    print(f"echo '{request_json}' | ./bin/vagrant-mcp-server")
    
    # Execute the command with the server if requested
    if execute:
        try:
            print("\nExecuting against MCP server:")
            proc = subprocess.Popen(
                ["./bin/vagrant-mcp-server"], 
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            stdout, stderr = proc.communicate(input=request_json, timeout=30)
            print("\nResponse:")
            if stderr:
                print("STDERR:")
                print(stderr)
            print("STDOUT:")
            print(stdout)
        except subprocess.TimeoutExpired:
            proc.kill()
            print("Error: Command timed out")
        except Exception as e:
            print(f"Error executing command: {e}")

if __name__ == "__main__":
    main()
