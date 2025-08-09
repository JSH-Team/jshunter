package start

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jsh-team/jshunter/internal/config"
	"github.com/jsh-team/jshunter/internal/db"
)

var (
	storageDir string
)

// StartCmd representa el comando para iniciar la aplicación
var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start JSHunter server",
	Run: func(cmd *cobra.Command, args []string) {
		config.InitializeBinaryPaths()
		if err := config.RunInstallationSteps(); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			os.Exit(1)
		}

		// Initialize binary paths to use cached binaries

		// Setup target storage configuration
		if err := config.SetupTargetStorage(config.Target, storageDir); err != nil {
			fmt.Printf("Failed to setup target storage: %v\n", err)
			os.Exit(0)
		}

		// Initialize database
		db.RunDB()
	},
}

func init() {
	StartCmd.Flags().IntVarP(&config.Port, "port", "p", config.DefaultPort, "Port to run the server")
	StartCmd.Flags().StringVarP(&config.Target, "target", "t", "", "Target Name")
	StartCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory for target data")
	StartCmd.Flags().BoolVar(&config.MobileExtractionEnabled, "mobile", false, "Enable mobile extraction")
	StartCmd.Flags().BoolVar(&config.ForceInstallation, "force", false, "Force installation")

	StartCmd.MarkFlagRequired("target")
}
