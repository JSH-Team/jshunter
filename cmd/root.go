package cmd

import (
	"fmt"
	"github.com/jsh-team/jshunter/cmd/start"
	"github.com/jsh-team/jshunter/cmd/targets"
	"github.com/jsh-team/jshunter/internal/config"

	"github.com/spf13/cobra"
)

var (
	// Version information
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"

	rootCmd = &cobra.Command{
		Use:   "jshunter",
		Short: "A tool for analyzing JavaScript files",
		Long: `JSHunter is a tool for analyzing JavaScript files.
This application is a tool to analyze JavaScript files.`,
		Version: version,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("JSHunter %s\n", version)
			fmt.Printf("Build time: %s\n", buildTime)
			fmt.Printf("Git commit: %s\n", gitCommit)
		},
	}
)

// SetVersion sets the version information
func SetVersion(v, bt, gc string) {
	version = v
	buildTime = bt
	gitCommit = gc
	rootCmd.Version = v
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Configuración de comandos
	startCmd := start.StartCmd
	targetsCmd := targets.TargetsCmd
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(targetsCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	config.LoadConfig()
}
