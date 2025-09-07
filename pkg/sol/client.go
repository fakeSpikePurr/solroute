package sol

import (
	"context"

	"github.com/gagliardetto/solana-go/rpc"
)

// Client represents a Solana client that handles both RPC and WebSocket connections
type Client struct {
	rpcClient   *rpc.Client
	jitoClient  *JitoClient
	rateLimiter *RateLimiter
}

// NewClient creates a new Solana client with custom rate limiting
func NewClient(ctx context.Context, endpoint, jitoEndpoint string, reqLimitPerSecond int) (*Client, error) {
	c := &Client{
		rpcClient:   rpc.New(endpoint),
		rateLimiter: NewRateLimiter(reqLimitPerSecond),
	}

	if jitoEndpoint != "" {
		jitoClient, err := NewJitoClient(ctx, jitoEndpoint)
		if err == nil {
			c.jitoClient = jitoClient
		}
	}
	return c, nil
}
