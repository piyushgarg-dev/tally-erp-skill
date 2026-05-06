package tally

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	url string
	hc  *http.Client
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url: url,
		hc:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Post(ctx context.Context, body string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return string(b), fmt.Errorf("tally returned HTTP %d", resp.StatusCode)
	}
	return string(b), nil
}
