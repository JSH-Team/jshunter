package extraction

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"jshunter/internal/utils/logger"
	urlutils "jshunter/internal/utils/url"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// BrowserExtractor provides a clean, focused approach to JavaScript extraction
type BrowserExtractor struct {
	browser     *rod.Browser
	browserURL  string
	mutex       sync.RWMutex
	timeout     time.Duration
	pageTimeout time.Duration
}

type ExtractionOptions struct {
	Headers     map[string]string
	Mobile      bool
	Timeout     time.Duration
	PageTimeout time.Duration
}

type JSResource struct {
	URL         string `json:"url"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Source      string `json:"source"` // "network", "dom", "inline"
}

// NewBrowserExtractor creates a new browser extractor
func NewBrowserExtractor() *BrowserExtractor {
	return &BrowserExtractor{
		timeout:     120 * time.Second,
		pageTimeout: 30 * time.Second,
	}
}

// Initialize sets up the browser instance
func (e *BrowserExtractor) Initialize() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.browser != nil {
		return nil
	}

	launcher := launcher.New().
		Headless(true).
		NoSandbox(true).
		Set("disable-extensions").
		Set("disable-default-apps").
		Set("disable-dev-shm-usage").
		Set("disable-gpu").
		Set("window-size", "1366,768")

	var err error
	e.browserURL, err = launcher.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	e.browser = rod.New().ControlURL(e.browserURL)
	if err := e.browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	return nil
}

// Close shuts down the browser
func (e *BrowserExtractor) Close() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.browser != nil {
		e.browser.Close()
		e.browser = nil
	}
}

// ExtractJavaScript extracts JavaScript resources from a URL
func (e *BrowserExtractor) ExtractJavaScript(url string, options ExtractionOptions) (string, []JSResource, error) {
	if err := e.Initialize(); err != nil {
		return "", nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	// Set timeout from options
	timeout := e.timeout
	if options.Timeout > 0 {
		timeout = options.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logger.Info("Starting extraction for %s", url)

	// Create new page
	page, err := e.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return "", nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// Set mobile viewport if requested
	if options.Mobile {
		if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:             430,
			Height:            932,
			DeviceScaleFactor: 3,
			Mobile:            true,
		}); err != nil {
			logger.Error("Failed to set mobile viewport: %v", err)
		}
	}

	// Collect JavaScript resources
	jsResources := make([]JSResource, 0)
	var resourcesMutex sync.Mutex

	// Set up request interception for network resources
	router := page.HijackRequests()
	defer func() {
		if err := router.Stop(); err != nil {
			logger.Error("Failed to stop router: %v", err)
		}
	}()

	router.MustAdd("*", func(hijack *rod.Hijack) {
		requestURL := hijack.Request.URL().String()

		// Don't intercept main page
		if requestURL == url {
			hijack.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// Apply custom headers
		req := hijack.Request.Req()
		e.setRequestHeaders(req, options.Headers)

		// Load response
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		if err := hijack.LoadResponse(client, true); err != nil {
			// Solo fallar silenciosamente, no hacer logging de errores de red
			hijack.Response.Fail(proto.NetworkErrorReasonFailed)
			return
		}

		// Extract JavaScript content
		contentType := hijack.Response.Headers().Get("Content-Type")
		if e.isJavaScriptResource(contentType, requestURL) {
			body := hijack.Response.Body()
			if len(body) > 0 {
				resourcesMutex.Lock()
				jsResources = append(jsResources, JSResource{
					URL:         requestURL,
					Content:     body,
					ContentType: "application/javascript",
					Source:      "network",
				})
				resourcesMutex.Unlock()
			}
		}
	})

	// Start router
	go router.Run()
	time.Sleep(100 * time.Millisecond) // Brief delay for router to start

	// Set user agent if provided
	if userAgent, exists := options.Headers["User-Agent"]; exists {
		if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: userAgent,
		}); err != nil {
			logger.Error("Failed to set user agent: %v", err)
		}
	}

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		return "", nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for initial content to load (no WaitLoad, just time-based)
	time.Sleep(3 * time.Second)

	// Wait for additional resources
	time.Sleep(5 * time.Second)

	// Extract HTML content
	htmlContent, err := page.HTML()
	if err != nil {
		logger.Error("Failed to get HTML content: %v", err)
		htmlContent = ""
	}

	// Extract DOM scripts
	domScripts := e.extractDOMScripts(page, ctx, url)
	resourcesMutex.Lock()
	jsResources = append(jsResources, domScripts...)
	resourcesMutex.Unlock()

	// Extract inline scripts
	inlineScripts, err := ExtractInlineJavaScript(htmlContent, url)
	if err != nil {
		logger.Error("Failed to extract inline scripts: %v", err)
	}
	resourcesMutex.Lock()
	for _, script := range inlineScripts {
		jsURL, err := GenerateInlineJSURL(url, script.Index)
		if err != nil {
			logger.Error("Failed to generate inline JS URL: %v", err)
			continue
		}
		jsResources = append(jsResources, JSResource{
			URL:         jsURL,
			Content:     script.Content,
			ContentType: "application/javascript",
			Source:      "inline",
		})
	}
	resourcesMutex.Unlock()

	logger.Info("Successfully extracted %d JavaScript resources from %s", len(jsResources), url)
	return htmlContent, jsResources, nil
}

// setRequestHeaders applies custom headers to HTTP requests
func (e *BrowserExtractor) setRequestHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
}

// isJavaScriptResource checks if content type indicates JavaScript
func (e *BrowserExtractor) isJavaScriptResource(contentType, url string) bool {
	// Exclude JSON explicitly
	if strings.Contains(strings.ToLower(contentType), "json") ||
		strings.HasSuffix(strings.ToLower(url), ".json") {
		return false
	}

	return strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "application/javascript") ||
		strings.Contains(contentType, "text/javascript") ||
		strings.HasSuffix(url, ".js")
}

// extractDOMScripts extracts external scripts from DOM
func (e *BrowserExtractor) extractDOMScripts(page *rod.Page, ctx context.Context, baseURL string) []JSResource {
	var resources []JSResource

	elements, err := page.Elements("script[src]")
	if err != nil {
		return resources
	}

	for _, el := range elements {
		src, err := el.Attribute("src")
		if err != nil || src == nil || *src == "" {
			continue
		}

		scriptURL := urlutils.NormalizeURL(*src, baseURL)
		content, err := page.GetResource(scriptURL)
		if err == nil && len(content) > 0 {
			resources = append(resources, JSResource{
				URL:         scriptURL,
				Content:     string(content),
				ContentType: "application/javascript",
				Source:      "dom",
			})
		}
	}

	return resources
}

// ExtractJavaScriptFromURL is a convenience function for one-off extractions
func ExtractJavaScriptFromURL(url string, headers map[string]string) ([]JSResource, error) {
	extractor := NewBrowserExtractor()
	defer extractor.Close()

	options := ExtractionOptions{
		Headers:     headers,
		Timeout:     60 * time.Second,
		PageTimeout: 20 * time.Second,
	}

	_, jsResources, err := extractor.ExtractJavaScript(url, options)
	return jsResources, err
}
