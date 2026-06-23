package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type amapClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func newAMapClient(baseURL, apiKey string, timeout time.Duration) *amapClient {
	if timeout <= 0 {
		timeout = defaultExternalAPITimeout
	}
	return &amapClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *amapClient) get(ctx context.Context, path string, query map[string]string, out any) error {
	if c == nil {
		return fmt.Errorf("amap client is nil")
	}
	if c.baseURL == "" {
		return fmt.Errorf("amap base url is empty")
	}
	if c.apiKey == "" {
		return fmt.Errorf("amap api key is empty")
	}

	values := url.Values{}
	values.Set("key", c.apiKey)
	values.Set("output", "json")
	for key, value := range query {
		if strings.TrimSpace(value) != "" {
			values.Set(key, value)
		}
	}

	reqURL := c.baseURL + "/" + strings.TrimLeft(path, "/") + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build amap request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call amap api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read amap response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("amap api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode amap response: %w", err)
	}
	if status, info := amapStatus(out); status != "" && status != "1" {
		if info == "" {
			info = "unknown amap error"
		}
		return fmt.Errorf("amap api error: %s", info)
	}
	return nil
}

func amapStatus(out any) (string, string) {
	data, err := json.Marshal(out)
	if err != nil {
		return "", ""
	}
	var base struct {
		Status string `json:"status"`
		Info   string `json:"info"`
	}
	if err := json.Unmarshal(data, &base); err != nil {
		return "", ""
	}
	return base.Status, base.Info
}
