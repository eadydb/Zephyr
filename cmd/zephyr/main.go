package main

import (
	"github.com/eadydb/zephyr/internal/cmd"
)

// main is the entry point of the Zephyr MCP server.
// All business logic has been moved to internal packages following Go best practices.
// This function only serves as the CLI entry point.
func main() {
	cmd.Execute()
}
