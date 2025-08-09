package analysis

import (
	"encoding/json"
	"fmt"
	"github.com/JSH-Team/JSHunter/internal/config"
	"os"
	"os/exec"
)

// NodeJSAnalyzerResult represents the structure returned by the Node.js analyzer
type NodeJSAnalyzerResult struct {
	URLs    []URLFinding    `json:"urls"`
	GQL     []GQLFinding    `json:"gql"`
	DomXSS  []DomXSSFinding `json:"domxss"`
	Events  []EventFinding  `json:"events"`
	HttpAPI []HttpFinding   `json:"httpapi"`
}

// URLFinding represents a URL finding from the Node.js analyzer
type URLFinding struct {
	Value    string                 `json:"value"`
	Line     int                    `json:"line"`
	Column   int                    `json:"column"`
	Type     string                 `json:"type"`
	Metadata map[string]interface{} `json:"metadata"`
}

// GQLFinding represents a GraphQL finding from the Node.js analyzer
type GQLFinding struct {
	Value  string `json:"value"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Type   string `json:"type"`
}

// DomXSSFinding represents a DOM XSS vulnerability finding
type DomXSSFinding struct {
	Value  string `json:"value"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Type   string `json:"type"`
}

// EventFinding represents an event-related finding
type EventFinding struct {
	Value  string `json:"value"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Type   string `json:"type"`
}

// HttpFinding represents an HTTP API call finding
type HttpFinding struct {
	Value   string      `json:"value"`
	Line    int         `json:"line"`
	Column  int         `json:"column"`
	Type    string      `json:"type"`
	URL     string      `json:"url,omitempty"`
	Method  string      `json:"method,omitempty"`
	Options interface{} `json:"options,omitempty"`
}

// Location represents the position of a finding in the source code
type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Finding represents a unified finding structure for database storage
type Finding struct {
	Type   string                 `json:"type"`
	Line   int                    `json:"line"`
	Column int                    `json:"column"`
	Value  string                 `json:"value"`
	Data   map[string]interface{} `json:"data"`
}

// NodeJSAnalyzer wraps the Node.js analyzer executable
type NodeJSAnalyzer struct {
	analyzerPath string
}

// NewNodeJSAnalyzer creates a new Node.js analyzer instance
func NewNodeJSAnalyzer() (*NodeJSAnalyzer, error) {
	// Use the analyzer path from configuration
	analyzerPath := config.AnalyzerBinaryPath
	if analyzerPath == "" {
		return nil, fmt.Errorf("analyzer binary path not configured")
	}

	// Check if the binary exists
	if _, err := os.Stat(analyzerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("analyzer binary not found at: %s", analyzerPath)
	}

	return &NodeJSAnalyzer{
		analyzerPath: analyzerPath,
	}, nil
}

// AnalyzeFile performs analysis on a JavaScript file using the Node.js analyzer
func (n *NodeJSAnalyzer) AnalyzeFile(filePath string) ([]Finding, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Run the Node.js analyzer with the file path
	cmd := exec.Command(n.analyzerPath, filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run Node.js analyzer: %w", err)
	}

	// Parse the JSON output
	var result NodeJSAnalyzerResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse analyzer output: %w", err)
	}

	// Convert to unified Finding format
	findings := n.convertToFindings(result)

	return findings, nil
}

// convertToFindings converts Node.js analyzer results to unified Finding format
func (n *NodeJSAnalyzer) convertToFindings(result NodeJSAnalyzerResult) []Finding {
	var findings []Finding

	// Convert URL findings
	for _, urlFinding := range result.URLs {
		// Ensure line is at least 1 (PocketBase requires non-zero values)
		line := urlFinding.Line
		if line <= 0 {
			line = 1
		}

		finding := Finding{
			Type:   urlFinding.Type,
			Line:   line,
			Column: urlFinding.Column,
			Value:  urlFinding.Value,
			Data: map[string]interface{}{
				"finding_category": "url",
				"metadata":         urlFinding.Metadata,
				"original_line":    urlFinding.Line, // Keep original line for reference
			},
		}
		findings = append(findings, finding)
	}

	// Convert GraphQL findings
	for _, gqlFinding := range result.GQL {
		// Ensure line is at least 1 (PocketBase requires non-zero values)
		line := gqlFinding.Line
		if line <= 0 {
			line = 1
		}

		finding := Finding{
			Type:   gqlFinding.Type,
			Line:   line,
			Column: gqlFinding.Column,
			Value:  gqlFinding.Value,
			Data: map[string]interface{}{
				"finding_category": "graphql",
				"original_line":    gqlFinding.Line, // Keep original line for reference
			},
		}
		findings = append(findings, finding)
	}

	// Convert DOM XSS findings
	for _, domxssFinding := range result.DomXSS {
		// Ensure line is at least 1 (PocketBase requires non-zero values)
		line := domxssFinding.Line
		if line <= 0 {
			line = 1
		}

		finding := Finding{
			Type:   domxssFinding.Type,
			Line:   line,
			Column: domxssFinding.Column,
			Value:  domxssFinding.Value,
			Data: map[string]interface{}{
				"finding_category": "domxss",
				"security_risk":    "high",
				"original_line":    domxssFinding.Line, // Keep original line for reference
			},
		}
		findings = append(findings, finding)
	}

	// Convert Event findings
	for _, eventFinding := range result.Events {
		// Ensure line is at least 1 (PocketBase requires non-zero values)
		line := eventFinding.Line
		if line <= 0 {
			line = 1
		}

		finding := Finding{
			Type:   eventFinding.Type,
			Line:   line,
			Column: eventFinding.Column,
			Value:  eventFinding.Value,
			Data: map[string]interface{}{
				"finding_category": "event",
				"original_line":    eventFinding.Line, // Keep original line for reference
			},
		}
		findings = append(findings, finding)
	}

	// Convert HTTP API findings
	for _, httpFinding := range result.HttpAPI {
		// Ensure line is at least 1 (PocketBase requires non-zero values)
		line := httpFinding.Line
		if line <= 0 {
			line = 1
		}

		data := map[string]interface{}{
			"finding_category": "httpapi",
			"original_line":    httpFinding.Line, // Keep original line for reference
		}

		// Add optional fields if they exist
		if httpFinding.URL != "" {
			data["url"] = httpFinding.URL
		}
		if httpFinding.Method != "" {
			data["method"] = httpFinding.Method
		}
		if httpFinding.Options != nil {
			data["options"] = httpFinding.Options
		}

		finding := Finding{
			Type:   httpFinding.Type,
			Line:   line,
			Column: httpFinding.Column,
			Value:  httpFinding.Value,
			Data:   data,
		}
		findings = append(findings, finding)
	}

	return findings
}

// AnalyzeFile is the main entry point for analyzing a JavaScript file
func AnalyzeFile(filePath string) ([]Finding, error) {
	analyzer, err := NewNodeJSAnalyzer()
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	return analyzer.AnalyzeFile(filePath)
}
