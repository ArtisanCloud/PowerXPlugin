package sts

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrMissingConfig = errors.New("sts client_id/client_secret/token_endpoint are required")

type Config struct {
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
	TTL           time.Duration
}

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
}

type Client struct {
	cfg Config
	mu  sync.Mutex
	now func() time.Time
	tok Token
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.TokenEndpoint) == "" || strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, ErrMissingConfig
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}
	return &Client{cfg: cfg, now: time.Now}, nil
}

func (c *Client) Token(ctx context.Context) (Token, error) {
	if c == nil {
		return Token{}, ErrMissingConfig
	}
	select {
	case <-ctx.Done():
		return Token{}, ctx.Err()
	default:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.tok.AccessToken != "" && now.Before(c.tok.ExpiresAt.Add(-30*time.Second)) {
		return c.tok, nil
	}
	c.tok = Token{
		AccessToken: "sts_" + c.cfg.ClientID,
		ExpiresAt:   now.Add(c.cfg.TTL),
	}
	return c.tok, nil
}

func (c *Client) HasConfig() bool {
	return c != nil && c.cfg.TokenEndpoint != "" && c.cfg.ClientID != "" && c.cfg.ClientSecret != ""
}
