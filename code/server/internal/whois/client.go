package whois

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Status string

const (
	Available Status = "AVAILABLE"
	Taken     Status = "TAKEN"
	Error     Status = "ERROR"
)

type Client struct {
	baseURL, apiKey string
	http            *http.Client
	timeout         time.Duration
	retries         int
}

func New(base, key string, timeout time.Duration, retries int) *Client {
	return &Client{strings.TrimRight(base, "/"), key, &http.Client{Timeout: timeout + 2*time.Second}, timeout, retries}
}

func (c *Client) Check(ctx context.Context, domain string) (Status, *string) {
	var last error
	for attempt := 0; attempt <= c.retries; attempt++ {
		status, err, retry := c.checkOnce(ctx, domain)
		if err == nil {
			return status, nil
		}
		last = err
		if !retry || attempt == c.retries {
			break
		}
		select {
		case <-ctx.Done():
			return Error, str(ctx.Err().Error())
		case <-time.After(time.Duration(1<<attempt) * 150 * time.Millisecond):
		}
	}
	if last == nil {
		last = errors.New("availability check failed")
	}
	return Error, str("Availability check unavailable")
}
func (c *Client) checkOnce(parent context.Context, domain string) (Status, error, bool) {
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	defer cancel()
	u := c.baseURL + "/domain/availability?domain=" + url.QueryEscape(domain) + "&apiKey=" + url.QueryEscape(c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Error, err, false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Error, err, true
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Error, fmt.Errorf("upstream status %d", resp.StatusCode), resp.StatusCode == 408 || resp.StatusCode == 429 || resp.StatusCode >= 500
	}
	var payload struct {
		Availability       string `json:"availability"`
		DomainAvailability string `json:"domain_availability"`
		DomainAvailable    *bool  `json:"domainAvailability"`
		Available          any    `json:"available"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return Error, err, false
	}
	v := strings.ToUpper(strings.TrimSpace(payload.Availability))
	if v == "" {
		v = strings.ToUpper(strings.TrimSpace(payload.DomainAvailability))
	}
	if v == "" {
		if payload.DomainAvailable != nil {
			if *payload.DomainAvailable {
				return Available, nil, false
			}
			return Taken, nil, false
		}
		if b, ok := payload.Available.(bool); ok {
			if b {
				return Available, nil, false
			}
			return Taken, nil, false
		}
	}
	switch v {
	case "AVAILABLE":
		return Available, nil, false
	case "UNAVAILABLE", "TAKEN":
		return Taken, nil, false
	default:
		return Error, errors.New("unknown availability response"), false
	}
}
func str(s string) *string { return &s }
