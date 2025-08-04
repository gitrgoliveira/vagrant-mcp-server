#!/bin/bash

# Simple test script for MCP exec_in_vm tool
echo "Hello from the MCP test script!"
echo "Current directory: $(pwd)"
echo "Files in current directory:"
ls -la
echo "Environment variables:"
env | sort
echo "Script execution complete!"
