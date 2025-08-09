package targets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jshunter/internal/config"

	"github.com/spf13/cobra"
)

type TargetInfo struct {
	Name       string
	StorageDir string
	IsActive   bool
	Exists     bool
	Size       string
	LastUsed   string
	DBExists   bool
	FilesCount int
}

func listTargets() error {
	// Load config to get targets
	config.LoadConfig()

	if len(config.GlobalConfig.Targets) == 0 {
		fmt.Println("No targets configured")
		return nil
	}

	// Print table header
	fmt.Printf("%-15s %-10s %-8s %s\n", "TARGET", "SIZE", "FILES", "PATH")
	fmt.Println(strings.Repeat("-", 80))

	for name, targetConfig := range config.GlobalConfig.Targets {
		if targetConfig.StorageDir != "" {
			if _, err := os.Stat(targetConfig.StorageDir); err == nil {
				size := formatSize(calculateDirSize(targetConfig.StorageDir))
				filesPath := filepath.Join(targetConfig.StorageDir, "files")
				fileCount := 0
				if _, err := os.Stat(filesPath); err == nil {
					fileCount = countFiles(filesPath)
				}
				fmt.Printf("%-15s %-10s %-8d %s\n", name, size, fileCount, targetConfig.StorageDir)
			} else {
				fmt.Printf("%-15s %-10s %-8s %s\n", name, "-", "-", targetConfig.StorageDir+" (not found)")
			}
		}
	}

	return nil
}

func calculateDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Hour {
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d hrs ago", int(diff.Hours()))
	} else if diff < 7*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	} else {
		return t.Format("2006-01-02")
	}
}

func countFiles(path string) int {
	count := 0
	filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

var TargetsCmd = &cobra.Command{
	Use:   "targets",
	Short: "List JSHunter targets",
	Long:  `List all configured JSHunter targets with their status and storage information.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listTargets(); err != nil {
			fmt.Printf("Error listing targets: %v\n", err)
			os.Exit(1)
		}
	},
}
