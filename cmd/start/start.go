package start

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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
	Long:  `Start JSHunter server`,
	Run: func(cmd *cobra.Command, args []string) {
		config.InitializeBinaryPaths()
		if err := config.RunInstallationSteps(); err != nil {
			fmt.Printf("Installation failed: %v\n", err)
			os.Exit(1)
		}

		// Setup target storage configuration
		if err := config.SetupTargetStorage(config.Target, storageDir); err != nil {
			fmt.Printf("Failed to setup target storage: %v\n", err)
			os.Exit(0)
		}

		// Initialize database
		db.RunDB()
	},
}

// Custom help function
func customHelpFunc(cmd *cobra.Command, args []string) {
	fmt.Printf("%s\n\n", cmd.Long)
	fmt.Printf("Usage:\n  %s\n\n", cmd.UseLine())

	fmt.Println("Flags:")
	fmt.Println("CONFIGURATION:")

	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "help" {
			return
		}
		if flag.Annotations != nil {
			if _, exists := flag.Annotations["group"]; exists {
				return
			}
		}

		short := ""
		if flag.Shorthand != "" {
			short = fmt.Sprintf(", -%s", flag.Shorthand)
		}

		defaultVal := ""
		if flag.DefValue != "" && flag.DefValue != "false" {
			defaultVal = fmt.Sprintf(" (default %s)", flag.DefValue)
		}

		fmt.Printf("   --%s%s %s    %s%s\n", flag.Name, short, flag.Value.Type(), flag.Usage, defaultVal)
	})

	fmt.Println("")
	fmt.Println("OPTIMIZATION:")

	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		if flag.Annotations == nil {
			return
		}
		if _, exists := flag.Annotations["group"]; !exists {
			return
		}

		short := ""
		if flag.Shorthand != "" {
			short = fmt.Sprintf(", -%s", flag.Shorthand)
		}

		defaultVal := ""
		if flag.DefValue != "" && flag.DefValue != "false" {
			defaultVal = fmt.Sprintf(" (default %s)", flag.DefValue)
		}

		fmt.Printf("   --%s%s %s    %s%s\n", flag.Name, short, flag.Value.Type(), flag.Usage, defaultVal)
	})
}

func init() {
	// Set custom help function
	StartCmd.SetHelpFunc(customHelpFunc)

	// Basic configuration flags
	StartCmd.Flags().IntVarP(&config.Port, "port", "p", config.DefaultPort, "Port to run the server")
	StartCmd.Flags().StringVarP(&config.Target, "target", "t", "", "Target Name")
	StartCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory for target data")
	StartCmd.Flags().BoolVar(&config.MobileExtractionEnabled, "mobile", false, "Enable mobile extraction")
	StartCmd.Flags().BoolVar(&config.ForceInstallation, "force", false, "Force installation")

	// Concurrency configuration flags
	StartCmd.Flags().IntVarP(&config.MaxConcurrentBrowsers, "concurrent-browsers", "b", config.MaxConcurrentBrowsers, "Maximum concurrent browser instances for extraction")
	StartCmd.Flags().IntVarP(&config.MaxConcurrentPrettify, "concurrent-prettify", "r", config.MaxConcurrentPrettify, "Maximum concurrent prettify workers")
	StartCmd.Flags().IntVarP(&config.MaxConcurrentSourcemaps, "concurrent-sourcemaps", "m", config.MaxConcurrentSourcemaps, "Maximum concurrent sourcemap workers")
	StartCmd.Flags().IntVarP(&config.MaxConcurrentAnalysis, "concurrent-analysis", "a", config.MaxConcurrentAnalysis, "Maximum concurrent analysis workers")
	StartCmd.Flags().IntVarP(&config.MaxConcurrentDechunker, "concurrent-dechunker", "d", config.MaxConcurrentDechunker, "Maximum concurrent dechunker workers")

	// Group flags using annotations for custom help formatting
	StartCmd.Flags().Lookup("concurrent-browsers").Annotations = map[string][]string{"group": {"OPTIMIZATION"}}
	StartCmd.Flags().Lookup("concurrent-prettify").Annotations = map[string][]string{"group": {"OPTIMIZATION"}}
	StartCmd.Flags().Lookup("concurrent-sourcemaps").Annotations = map[string][]string{"group": {"OPTIMIZATION"}}
	StartCmd.Flags().Lookup("concurrent-analysis").Annotations = map[string][]string{"group": {"OPTIMIZATION"}}
	StartCmd.Flags().Lookup("concurrent-dechunker").Annotations = map[string][]string{"group": {"OPTIMIZATION"}}

	StartCmd.MarkFlagRequired("target")
}
