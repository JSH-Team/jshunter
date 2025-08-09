package extraction

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// InlineJS represents an extracted inline JavaScript
type InlineJS struct {
	Content string
	Index   int
}

// ExtractInlineJavaScript extracts all inline JavaScript from HTML content
func ExtractInlineJavaScript(htmlContent, baseURL string) ([]InlineJS, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var inlineScripts []InlineJS
	index := 1

	// Traverse the HTML tree to find script tags
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			// Check if this is an external script (has src attribute)
			hasSrc := false
			scriptType := ""

			for _, attr := range n.Attr {
				if attr.Key == "src" {
					hasSrc = true
					break
				}
				if attr.Key == "type" {
					scriptType = strings.ToLower(strings.TrimSpace(attr.Val))
				}
			}

			// Skip external scripts
			if hasSrc {
				return
			}

			// Skip non-JavaScript script tags
			if scriptType != "" {
				if scriptType == "application/ld+json" ||
					scriptType == "application/json" ||
					scriptType == "text/css" ||
					scriptType == "text/template" ||
					(!strings.Contains(scriptType, "javascript") && scriptType != "text/javascript") {
					return
				}
			}

			// Extract text content from script tag
			var content strings.Builder
			var extractText func(*html.Node)
			extractText = func(node *html.Node) {
				if node.Type == html.TextNode {
					content.WriteString(node.Data)
				}
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					extractText(child)
				}
			}

			for child := n.FirstChild; child != nil; child = child.NextSibling {
				extractText(child)
			}

			scriptContent := strings.TrimSpace(content.String())
			if scriptContent != "" {
				inlineScripts = append(inlineScripts, InlineJS{
					Content: scriptContent,
					Index:   index,
				})
				index++
			}
		}

		// Continue traversing child nodes
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	traverse(doc)
	return inlineScripts, nil
}

// GenerateInlineJSURL creates a URL for inline JavaScript based on the base URL and index
func GenerateInlineJSURL(baseURL string, index int) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Create a filename like inline_1.js, inline_2.js, etc.
	filename := fmt.Sprintf("inline_%d.js", index)

	// If the URL has a path, append to it, otherwise use root
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		parsedURL.Path = "/" + filename
	} else {
		// Remove trailing slash if present and append filename
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/" + filename
	}
	return parsedURL.String(), nil
}
