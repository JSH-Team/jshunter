package config

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	Port         int
	Target       string
	StorageDir   string // Single storage directory for both DB and files
	GlobalConfig Config

	// Binary paths - populated during initialization
	AnalyzerBinaryPath   string
	PrettifierBinaryPath string
	DechunkerBinaryPath  string
	ForceInstallation    bool

	// Browser worker pool configuration (extraction)
	MaxConcurrentBrowsers = 4   // Maximum concurrent browser instances
	BrowserWorkerTimeout  = 90  // Timeout in seconds for browser processing
	QueueBufferSize       = 100 // Size of extraction processing queue buffer

	// Prettify worker pool configuration
	MaxConcurrentPrettify = 8   // Maximum concurrent prettify workers (CPU intensive)
	PrettifyQueueSize     = 400 // Size of prettify processing queue buffer

	// Sourcemap worker pool configuration
	MaxConcurrentSourcemaps = 4   // Maximum concurrent sourcemap workers (I/O intensive)
	SourcemapQueueSize      = 400 // Size of sourcemap processing queue buffer

	// Analysis worker pool configuration
	MaxConcurrentAnalysis = 6   // Maximum concurrent analysis workers (CPU intensive)
	AnalysisQueueSize     = 400 // Size of analysis processing queue buffer

	// Dechunker worker pool configuration
	MaxConcurrentDechunker = 4   // Maximum concurrent dechunker workers (CPU intensive)
	DechunkerQueueSize     = 400 // Size of dechunker processing queue buffer

	// Mobile extraction configuration
	MobileExtractionEnabled = false // Whether mobile extraction is enabled
)

var DefaultConfig = Config{
	Targets: make(map[string]TargetConfig),
}

type Config struct {
	Targets map[string]TargetConfig `mapstructure:"targets" yaml:"targets"`

	// Browser processing configuration (extraction)
	MaxConcurrentBrowsers int `mapstructure:"max_concurrent_browsers" yaml:"max_concurrent_browsers"`
	WorkerPoolSize        int `mapstructure:"worker_pool_size" yaml:"worker_pool_size"`
	BrowserTimeout        int `mapstructure:"browser_timeout" yaml:"browser_timeout"`
}

type TargetConfig struct {
	StorageDir string `mapstructure:"storage_dir" yaml:"storage_dir"`
}

// GetDbPath returns the database path for the current target
func GetDbPath() string {
	if StorageDir != "" {
		return filepath.Join(StorageDir, "db")
	}
	return ""
}

// GetFilesPath returns the files storage path for the current target
func GetFilesPath() string {
	if StorageDir != "" {
		return filepath.Join(StorageDir, "files")
	}
	return ""
}

func GetLibsDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "jshunter", "libs")
}

// InitializeBinaryPaths sets up the binary paths based on the current OS and architecture
func InitializeBinaryPaths() {
	libsDir := GetLibsDirectory()

	AnalyzerBinaryPath = filepath.Join(libsDir, GetAnalyzerBinaryName())
	PrettifierBinaryPath = filepath.Join(libsDir, GetPrettifierBinaryName())
	DechunkerBinaryPath = filepath.Join(libsDir, GetDechunkerBinaryName())
}

// getAnalyzerBinaryName returns the analyzer binary name for the current platform
func GetAnalyzerBinaryName() string {
	if runtime.GOOS == "windows" {
		return "analyzer.exe"
	}
	return "analyzer"
}

// getPrettifierBinaryName returns the prettifier binary name for the current platform
func GetPrettifierBinaryName() string {
	if runtime.GOOS == "windows" {
		return "prettifier.exe"
	}
	return "prettifier"
}

// getDechunkerBinaryName returns the dechunker binary name for the current platform
func GetDechunkerBinaryName() string {
	if runtime.GOOS == "windows" {
		return "dechunker.exe"
	}
	return "dechunker"
}
