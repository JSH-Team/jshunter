package html

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// GenerateHTMLHash normalizes an HTML string and returns a SHA-256 hash of the meaningful content.
func GenerateHTMLHash(htmlContent string) (string, error) {
	// Parse HTML into a goquery document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	// Remove comments
	doc.Find("comment").Remove()

	// Remove dynamic elements: scripts, styles, and meta tags
	doc.Find("script, style, meta").Remove()

	// Remove all attributes except essential ones (e.g., keep href, src for functionality)
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		attrs := s.Nodes[0].Attr
		s.Nodes[0].Attr = nil // Clear all attributes
		for _, attr := range attrs {
			// Keep only essential attributes like href, src
			if attr.Key == "href" || attr.Key == "src" {
				s.SetAttr(attr.Key, attr.Val)
			}
		}
	})

	// Normalize text content (optional: remove dynamic text like dates or nonces)
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		// Remove common WordPress dynamic patterns (e.g., nonces, timestamps)
		text = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).ReplaceAllString(text, "") // UUIDs
		text = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T?\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})?`).ReplaceAllString(text, "")     // ISO dates
		text = regexp.MustCompile(`nonce-[0-9a-f]+`).ReplaceAllString(text, "")                                              // WordPress nonces
		s.Nodes[0].FirstChild = &html.Node{
			Type: html.TextNode,
			Data: text,
		}
	})

	// Get normalized HTML
	normalized, err := doc.Html()
	if err != nil {
		return "", err
	}

	// Normalize whitespace: collapse multiple spaces, remove newlines
	normalized = strings.Join(strings.Fields(strings.TrimSpace(normalized)), " ")

	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:]), nil
}
