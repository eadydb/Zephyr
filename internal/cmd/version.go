package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build details and runtime information.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Version-specific flags
	versionCmd.Flags().BoolP("short", "s", false, "print only the version number")
}

func runVersion(cmd *cobra.Command, args []string) {
	short, _ := cmd.Flags().GetBool("short")

	if short {
		fmt.Println(serverVersion)
		return
	}

	fmt.Printf("Zephyr MCP Server\n")
	fmt.Printf("Version:    %s\n", serverVersion)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Compiler:   %s\n", runtime.Compiler)
}
