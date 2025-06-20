package cmd

import (
	"fmt"
	"os"

	"github.com/eadydb/zephyr/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Commands for validating, displaying, and managing Zephyr configuration.`,
}

// validateCmd represents the config validate subcommand
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  `Validate the configuration file for syntax errors and required fields.`,
	RunE:  runValidateConfig,
}

// showCmd represents the config show subcommand
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long:  `Display the current configuration with all defaults applied and environment variables resolved.`,
	RunE:  runShowConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(showCmd)

	// Config-specific flags
	showCmd.Flags().BoolP("raw", "r", false, "show raw configuration without formatting")
}

func runValidateConfig(cmd *cobra.Command, args []string) error {
	configPath := GetConfigFile()
	if configPath == "" {
		configPath = "config.yaml"
	}

	_, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		return err
	}

	fmt.Printf("Configuration file '%s' is valid\n", configPath)
	return nil
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	configPath := GetConfigFile()
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	raw, _ := cmd.Flags().GetBool("raw")

	if raw {
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}
		fmt.Print(string(data))
	} else {
		fmt.Printf("Configuration loaded from: %s\n\n", configPath)

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}
		fmt.Print(string(data))
	}

	return nil
}
