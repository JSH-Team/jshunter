package fetch

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/ratelimit"
)

type AssetFetcher interface {
	RateLimitedGet(ctx context.Context, url string) (string, bool, error)
	RateLimitedHead(ctx context.Context, url string) (string, bool, error)
	Request(ctx context.Context, url string, method string) (string, bool, error)
	RateLimitedGetWithContentType(ctx context.Context, url string) (string, string, bool, error)
}

type assetFetcherImpl struct {
	client      *http.Client
	rateLimiter ratelimit.Limiter
}

func NewAssetFetcher() *assetFetcherImpl {
	// taken from https://github.com/sweetbbak/go-cloudflare-bypass
	tlsConfig := http.DefaultTransport.(*http.Transport).TLSClientConfig

	c := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 30 * time.Second,
			DisableKeepAlives:   false,

			TLSClientConfig: &tls.Config{
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_AES_128_GCM_SHA256,
					tls.VersionTLS13,
					tls.VersionTLS10,
				},
				InsecureSkipVerify: true, // Disable certificate verification
			},
			DialTLS: func(network, addr string) (net.Conn, error) {
				return tls.Dial(network, addr, tlsConfig)
			},
		},
	}

	rateLimiter := ratelimit.New(30, ratelimit.Per(time.Minute))

	return &assetFetcherImpl{
		client:      c,
		rateLimiter: rateLimiter,
	}
}

func (s *assetFetcherImpl) RateLimitedGet(ctx context.Context, url string) (string, bool, error) {
	s.rateLimiter.Take()

	return s.Request(ctx, url, "GET")
}

func (s *assetFetcherImpl) RateLimitedGetWithContentType(ctx context.Context, url string) (string, string, bool, error) {
	s.rateLimiter.Take()

	return s.RequestWithContentType(ctx, url, "GET")
}

func (s *assetFetcherImpl) RateLimitedHead(ctx context.Context, url string) (string, bool, error) {
	s.rateLimiter.Take()

	return s.Request(ctx, url, "HEAD")
}

// Get is a regular HTTP get but handles GZIP and adds headers to avoid being detected as bot.
func (s *assetFetcherImpl) Request(ctx context.Context, url string, method string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", false, err
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "script")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")

	//for key, value := range headers {
	//	req.Header.Set(key, value)
	//}

	req.Header.Set("accept-encoding", "gzip")

	resp, err := s.client.Do(req)

	if err != nil {
		return "", false, nil
	}
	defer resp.Body.Close()

	// Read the entire response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, nil
	}

	// Check if the response is gzipped
	contentEncoding := resp.Header.Get("Content-Encoding")
	isGzipped := strings.Contains(contentEncoding, "gzip")

	// If not marked as gzipped, check for gzip magic number
	if !isGzipped {
		isGzipped = len(body) > 2 && body[0] == 0x1f && body[1] == 0x8b
	}

	// If gzipped, decompress
	if isGzipped {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return "", false, nil
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return "", false, nil
		}
		body = decompressed
	}

	return string(body), resp.StatusCode == http.StatusOK, nil
}

func (s *assetFetcherImpl) RequestWithContentType(ctx context.Context, url string, method string) (string, string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", "", false, err
	}

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-GB,en-US;q=0.9,en;q=0.8")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "script")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")

	req.Header.Set("accept-encoding", "gzip")

	resp, err := s.client.Do(req)

	if err != nil {
		return "", "", false, nil
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// Read the entire response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", contentType, false, nil
	}

	// Check if the response is gzipped
	contentEncoding := resp.Header.Get("Content-Encoding")
	isGzipped := strings.Contains(contentEncoding, "gzip")

	// If not marked as gzipped, check for gzip magic number
	if !isGzipped {
		isGzipped = len(body) > 2 && body[0] == 0x1f && body[1] == 0x8b
	}

	// If gzipped, decompress
	if isGzipped {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return "", contentType, false, nil
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return "", contentType, false, nil
		}
		body = decompressed
	}

	return string(body), contentType, resp.StatusCode == http.StatusOK, nil
}
