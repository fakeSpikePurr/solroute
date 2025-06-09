package sol

import (
	"context"
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// Client represents a Solana client that handles both RPC and WebSocket connections
type Client struct {
	RpcClient *rpc.Client
	WsClient  *ws.Client
}

// NewClient creates a new Solana client with both RPC and WebSocket connections
func NewClient(ctx context.Context, endpoint string) (*Client, error) {

	// Initialize WebSocket client
	wsEndpoint := strings.Replace(endpoint, "https://", "wss://", 1)
	wsClient, err := ws.Connect(ctx, wsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to establish WebSocket connection: %w", err)
	}

	return &Client{
		RpcClient: rpc.New(endpoint),
		WsClient:  wsClient,
	}, nil
}

// Close terminates all client connections
func (c *Client) Close() error {
	if c.WsClient != nil {
		c.WsClient.Close()
	}
	return nil
}
