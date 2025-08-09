package url

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

func RemoveQueryString(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, err
	}

	// Eliminar query string
	parsedURL.RawQuery = ""

	// También eliminar el fragmento (#)
	parsedURL.Fragment = ""

	return parsedURL.String(), nil
}

func ToAbsoluteURL(baseStr, inputStr string) (string, error) {
	u, err := url.Parse(inputStr)
	if err != nil {
		return "", err
	}

	// If the input is already an absolute URL, return it as-is
	if u.IsAbs() {
		return u.String(), nil
	}

	// Otherwise, resolve it against the base
	base, err := url.Parse(baseStr)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(u).String(), nil
}

// decodeDataURI handles data: URIs (inline sourcemaps)
func DecodeDataURI(dataURI string) ([]byte, error) {
	// Split the data URI into header and content parts
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URI format")
	}

	// Check if content is base64 encoded
	header := parts[0]
	isBase64 := strings.Contains(header, ";base64")

	if isBase64 {
		return base64.StdEncoding.DecodeString(parts[1])
	}

	// Return raw content if not base64 encoded
	return []byte(parts[1]), nil
}

func GetDomainFromUrl(rawUrl string) (string, error) {
	// Extract domain from URL
	parsedURL, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("error parsing URL '%s': %v", rawUrl, err)
	}

	domain := parsedURL.Host
	if domain == "" {
		return "", fmt.Errorf("no domain found in URL '%s'", rawUrl)
	}

	return domain, nil
}

func GetFileNameFromUrl(rawUrl string) (string, error) {
	// Extract domain from URL
	parsedURL, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("error parsing URL '%s': %v", rawUrl, err)
	}

	// Split the URL path to get the last part (filename)
	pathParts := strings.Split(parsedURL.Path, "/")
	if len(pathParts) == 0 {
		return "", fmt.Errorf("empty URL path '%s'", parsedURL.Path)
	}

	// Get the last part (filename)
	fileName := pathParts[len(pathParts)-1]
	if fileName == "" {
		fileName = "index.html"
	}
	// Truncate if the filename is too long
	if len(fileName) > 100 {
		fileName = fileName[:100]
	}
	return fileName, nil
}

func NormalizeURL(scriptURL, baseURL string) string {
	// Convert relative URLs to absolute
	if !strings.HasPrefix(scriptURL, "http") {
		if strings.HasPrefix(scriptURL, "//") {
			if strings.HasPrefix(baseURL, "https") {
				scriptURL = "https:" + scriptURL
			} else {
				scriptURL = "http:" + scriptURL
			}
		} else if strings.HasPrefix(scriptURL, "/") {
			// Root-relative URL
			parts := strings.SplitN(baseURL, "/", 4)
			if len(parts) >= 3 {
				scriptURL = parts[0] + "//" + parts[2] + scriptURL
			}
		} else {
			// Relative URL
			base := baseURL
			if strings.HasSuffix(base, "/") {
				base = strings.TrimSuffix(base, "/")
			}
			scriptURL = base + "/" + scriptURL
		}
	}
	return scriptURL
}
