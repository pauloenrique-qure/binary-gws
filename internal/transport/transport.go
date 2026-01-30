package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Config struct {
	APIURLs            []string
	TokenCurrent       string
	TokenGrace         string
	CABundlePath       string
	InsecureSkipVerify bool
	RequestTimeout     time.Duration
}

type Client struct {
	config     Config
	httpClient *http.Client
}

type RetryConfig struct {
	MaxRetries int
	Delays     []time.Duration
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries: 3,
	Delays:     []time.Duration{5 * time.Second, 10 * time.Second, 15 * time.Second},
}

type Sleeper interface {
	Sleep(d time.Duration)
}

type RealSleeper struct{}

func (RealSleeper) Sleep(d time.Duration) {
	time.Sleep(d)
}

func New(cfg Config) (*Client, error) {
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 10 * time.Second
	}
	if len(cfg.APIURLs) == 0 {
		return nil, fmt.Errorf("api_urls is required")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	if cfg.CABundlePath != "" {
		caCert, err := os.ReadFile(cfg.CABundlePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA bundle: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA bundle")
		}
		tlsConfig.RootCAs = caCertPool
	}

	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 2,
		},
	}

	return &Client{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

func (c *Client) SendHeartbeat(ctx context.Context, payload interface{}, retryConfig RetryConfig, sleeper Sleeper) error {
	if sleeper == nil {
		sleeper = RealSleeper{}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	tokens := []string{c.config.TokenCurrent}
	if c.config.TokenGrace != "" {
		tokens = append(tokens, c.config.TokenGrace)
	}

	var lastErr error
	for _, apiURL := range c.config.APIURLs {
		if apiURL == "" {
			continue
		}
		err := c.sendToURL(ctx, apiURL, jsonData, tokens, retryConfig, sleeper)
		if err == nil {
			return nil
		}
		if !shouldTryFallback(err) {
			return err
		}
		lastErr = err
	}

	return lastErr
}

func extractStatusCode(err error) int {
	if err == nil {
		return 0
	}
	var statusCode int
	fmt.Sscanf(err.Error(), "authentication failed: HTTP %d", &statusCode)
	if statusCode == 0 {
		fmt.Sscanf(err.Error(), "client error: HTTP %d", &statusCode)
	}
	if statusCode == 0 {
		fmt.Sscanf(err.Error(), "server error: HTTP %d", &statusCode)
	}
	return statusCode
}

func shouldTryFallback(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	statusCode := extractStatusCode(err)
	if statusCode >= 400 && statusCode < 500 {
		return false
	}
	return true
}

func (c *Client) sendToURL(ctx context.Context, apiURL string, jsonData []byte, tokens []string, retryConfig RetryConfig, sleeper Sleeper) error {
	var lastErr error
	for tokenIndex, token := range tokens {
		for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
			if attempt > 0 {
				delayIndex := attempt - 1
				if delayIndex >= len(retryConfig.Delays) {
					delayIndex = len(retryConfig.Delays) - 1
				}
				delay := retryConfig.Delays[delayIndex]
				sleeper.Sleep(delay)
			}

			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
			if err != nil {
				lastErr = fmt.Errorf("failed to create request: %w", err)
				continue
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("request failed: %w", err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				continue
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}

			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				if tokenIndex == 0 && c.config.TokenGrace != "" {
					lastErr = fmt.Errorf("authentication failed: HTTP %d", resp.StatusCode)
					break
				}
				return fmt.Errorf("authentication failed: HTTP %d", resp.StatusCode)
			}

			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return fmt.Errorf("client error: HTTP %d: %s", resp.StatusCode, string(body))
			}

			lastErr = fmt.Errorf("server error: HTTP %d: %s", resp.StatusCode, string(body))
		}

		if lastErr != nil {
			statusCode := extractStatusCode(lastErr)
			if statusCode == 401 || statusCode == 403 {
				continue
			}
		}
		break
	}

	return lastErr
}
