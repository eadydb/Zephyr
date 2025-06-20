package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	logLevel  string
	logFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zephyr",
	Short: "Zephyr MCP Server - A Model Context Protocol server with plugin support",
	Long: `Zephyr is a high-performance MCP (Model Context Protocol) server written in Go.
It supports multiple transport protocols (STDIO, SSE, HTTP) and provides a 
plugin architecture for extending functionality with custom tools.

Features:
  • Multi-protocol transport support (STDIO, SSE, HTTP)
  • Plugin-based architecture for custom tools
  • Built-in monitoring and metrics
  • Configuration via YAML files and environment variables
  • Graceful shutdown and resource management`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (text, json)")

	// Bind flags to viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("log.format", rootCmd.PersistentFlags().Lookup("log-format"))
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Environment variables
	viper.SetEnvPrefix("ZEPHYR")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

// GetConfigFile returns the config file path
func GetConfigFile() string {
	return cfgFile
}

// GetLogLevel returns the log level
func GetLogLevel() string {
	return logLevel
}

// GetLogFormat returns the log format
func GetLogFormat() string {
	return logFormat
}
