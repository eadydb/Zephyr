package cmd

import (
	"fmt"

	"github.com/eadydb/zephyr/internal/app"
	"github.com/spf13/cobra"
)

const (
	serverName    = "zephyr-mcp-server"
	serverVersion = "1.0.0"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the Zephyr MCP (Model Context Protocol) server with the specified configuration.

The server will:
  • Load plugins from the configured directories
  • Start the MCP server with the specified transport protocol
  • Begin listening for MCP requests
  • Start monitoring endpoints if enabled
  • Watch for configuration changes if hot reload is enabled

The server runs until interrupted with Ctrl+C or a termination signal.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Serve-specific flags
	serveCmd.Flags().String("transport", "", "transport protocol (stdio, sse, http)")
	serveCmd.Flags().String("host", "", "host address for network transports")
	serveCmd.Flags().Int("port", 0, "port number for network transports")
	serveCmd.Flags().Bool("monitoring", false, "enable monitoring endpoints")
	serveCmd.Flags().Bool("hot-reload", false, "enable configuration hot reload")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Check for hot reload flag
	hotReload, _ := cmd.Flags().GetBool("hot-reload")

	// Get CLI configuration
	opts := &app.AppOptions{
		ConfigPath:      GetConfigFile(),
		LogLevel:        GetLogLevel(),
		LogFormat:       GetLogFormat(),
		EnableHotReload: hotReload,
	}

	// Create and initialize application
	application, err := app.New(serverName, serverVersion, opts)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	// Override configuration with CLI flags if provided
	if err := applyServeFlags(cmd, application); err != nil {
		return fmt.Errorf("failed to apply CLI flags: %w", err)
	}

	// Run the application
	return application.Run()
}

func applyServeFlags(cmd *cobra.Command, app *app.App) error {
	config := app.GetConfig()

	// Apply transport protocol override
	if cmd.Flags().Changed("transport") {
		transport, _ := cmd.Flags().GetString("transport")
		config.Transport.Protocol = transport
	}

	// Apply host override
	if cmd.Flags().Changed("host") {
		host, _ := cmd.Flags().GetString("host")
		config.Transport.SSE.Host = host
		config.Transport.HTTP.Host = host
	}

	// Apply port override
	if cmd.Flags().Changed("port") {
		port, _ := cmd.Flags().GetInt("port")
		config.Transport.SSE.Port = port
		config.Transport.HTTP.Port = port
	}

	// Apply monitoring override
	if cmd.Flags().Changed("monitoring") {
		monitoring, _ := cmd.Flags().GetBool("monitoring")
		config.Monitoring.Enabled = monitoring
	}

	return nil
}
