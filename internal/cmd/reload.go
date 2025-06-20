package cmd

import (
	"fmt"
	"os"

	"github.com/eadydb/zephyr/internal/config"
	"github.com/spf13/cobra"
)

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload configuration",
	Long: `Reload configuration commands for the running MCP server.

Note: This command validates the configuration file but does not communicate
with a running server instance. For runtime configuration reloading, the server
must be started with the --hot-reload flag.`,
}

// configReloadCmd represents the config reload subcommand
var configReloadCmd = &cobra.Command{
	Use:   "config",
	Short: "Validate and test configuration reload",
	Long: `Validate the configuration file and test if it can be successfully reloaded.

This command:
  • Loads the configuration file
  • Validates syntax and required fields
  • Reports any issues that would prevent hot reload

This is useful for testing configuration changes before applying them
to a running server with hot reload enabled.`,
	RunE: runConfigReload,
}

func init() {
	rootCmd.AddCommand(reloadCmd)
	reloadCmd.AddCommand(configReloadCmd)

	// Reload-specific flags
	configReloadCmd.Flags().BoolP("verbose", "v", false, "show detailed configuration after reload test")
}

func runConfigReload(cmd *cobra.Command, args []string) error {
	configPath := GetConfigFile()
	if configPath == "" {
		configPath = "config.yaml"
	}

	fmt.Printf("Testing configuration reload from: %s\n", configPath)

	// Test loading the configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration reload test failed: %v\n", err)
		return err
	}

	fmt.Printf("✅ Configuration reload test successful\n")

	// Show verbose output if requested
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		fmt.Printf("\nConfiguration details:\n")
		fmt.Printf("  Server: %s v%s\n", cfg.Server.Name, cfg.Server.Version)
		fmt.Printf("  Transport: %s\n", cfg.Transport.Protocol)
		fmt.Printf("  Monitoring: %v (port %d)\n", cfg.Monitoring.Enabled, cfg.Monitoring.Port)
		fmt.Printf("  Plugins enabled: %d\n", countEnabledPlugins(cfg))

		if cfg.Server.Debug {
			fmt.Printf("  Debug mode: enabled\n")
		}
	}

	fmt.Printf("\n💡 To enable hot reload in the server, use: zephyr serve --hot-reload\n")
	return nil
}

// countEnabledPlugins counts the number of enabled plugins in the configuration
func countEnabledPlugins(cfg *config.Config) int {
	count := 0
	for _, tool := range cfg.Plugins.Tools {
		if tool.Enabled {
			count++
		}
	}
	return count
}
