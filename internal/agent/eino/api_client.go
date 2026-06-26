package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type amapClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	limiter    *externalAPILimiter
}

type externalAPILimiter struct {
	sem      chan struct{}
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func newExternalAPILimiter(concurrency, qps int) *externalAPILimiter {
	if concurrency <= 0 {
		concurrency = defaultExternalAPIConcurrency
	}
	if qps <= 0 {
		qps = defaultExternalAPIQPS
	}
	return &externalAPILimiter{
		sem:      make(chan struct{}, concurrency),
		interval: time.Second / time.Duration(qps),
	}
}

func (l *externalAPILimiter) wait(ctx context.Context) (func(), error) {
	if l == nil {
		return func() {}, nil
	}

	if l.interval > 0 {
		now := time.Now()
		l.mu.Lock()
		sendAt := now
		if l.next.After(now) {
			sendAt = l.next
		}
		l.next = sendAt.Add(l.interval)
		l.mu.Unlock()

		if delay := time.Until(sendAt); delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, nil
	}
}

func newAMapClient(baseURL, apiKey string, timeout time.Duration, limiters ...*externalAPILimiter) *amapClient {
	if timeout <= 0 {
		timeout = defaultExternalAPITimeout
	}
	var limiter *externalAPILimiter
	if len(limiters) > 0 {
		limiter = limiters[0]
	}
	return &amapClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter: limiter,
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
	release, err := c.limiter.wait(ctx)
	if err != nil {
		return fmt.Errorf("wait for amap rate limit: %w", err)
	}
	defer release()

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
	started := time.Now()
	log.Printf("[AMap API] call path=%s query=%s", strings.TrimLeft(path, "/"), safeAMapQuery(values))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[AMap API] return path=%s error=%v duration_ms=%d", strings.TrimLeft(path, "/"), err, time.Since(started).Milliseconds())
		return fmt.Errorf("call amap api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		log.Printf("[AMap API] return path=%s status=%d error=%v duration_ms=%d", strings.TrimLeft(path, "/"), resp.StatusCode, err, time.Since(started).Milliseconds())
		return fmt.Errorf("read amap response: %w", err)
	}
	log.Printf("[AMap API] return path=%s status=%d bytes=%d duration_ms=%d body=%s", strings.TrimLeft(path, "/"), resp.StatusCode, len(body), time.Since(started).Milliseconds(), string(body))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("amap api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode amap response: %w", err)
	}
	if status, info := amapStatus(out); status != "" && status != "1" {
		log.Printf("[AMap API] value path=%s amap_status=%s info=%s", strings.TrimLeft(path, "/"), status, info)
		if info == "" {
			info = "unknown amap error"
		}
		return fmt.Errorf("amap api error: %s", info)
	} else if status != "" {
		log.Printf("[AMap API] value path=%s amap_status=%s info=%s", strings.TrimLeft(path, "/"), status, info)
	}
	return nil
}

func safeAMapQuery(values url.Values) string {
	safe := url.Values{}
	for key, items := range values {
		if strings.EqualFold(key, "key") {
			continue
		}
		for _, item := range items {
			safe.Add(key, item)
		}
	}
	return safe.Encode()
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
