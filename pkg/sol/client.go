package sol

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// Client represents a Solana client that handles both RPC and WebSocket connections
type Client struct {
	rpcClient   *rpc.Client
	wsClient    *ws.Client
	jitoClient  *JitoClient
	rateLimiter *RateLimiter
}

// NewClient creates a new Solana client with custom rate limiting
func NewClient(ctx context.Context, endpoint, wsEndpoint, jitoEndpoint string, reqLimitPerSecond int) (*Client, error) {
	c := &Client{
		rpcClient:   rpc.New(endpoint),
		rateLimiter: NewRateLimiter(reqLimitPerSecond),
	}
	if wsEndpoint != "" {
		// Initialize WebSocket client
		wsClient, err := ws.Connect(ctx, wsEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to establish WebSocket connection: %w", err)
		}
		c.wsClient = wsClient
	}
	if jitoEndpoint != "" {
		jitoClient, err := NewJitoClient(ctx, jitoEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to create Jito client: %w", err)
		}
		c.jitoClient = jitoClient
	}
	return c, nil
}

// Close terminates all client connections
func (c *Client) Close() error {
	if c.wsClient != nil {
		c.wsClient.Close()
	}
	return nil
}
