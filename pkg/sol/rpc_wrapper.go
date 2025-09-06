package sol

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// RPC wrapper methods with rate limiting

// GetAccountInfo wraps the RPC call with rate limiting
func (c *Client) GetAccountInfo(ctx context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetAccountInfo(ctx, account)
}

// GetAccountInfoWithOpts wraps the RPC call with rate limiting
func (c *Client) GetAccountInfoWithOpts(ctx context.Context, account solana.PublicKey, opts *rpc.GetAccountInfoOpts) (*rpc.GetAccountInfoResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetAccountInfoWithOpts(ctx, account, opts)
}

// GetMultipleAccounts wraps the RPC call with rate limiting
func (c *Client) GetMultipleAccounts(ctx context.Context, accounts []solana.PublicKey) (*rpc.GetMultipleAccountsResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetMultipleAccounts(ctx, accounts...)
}

// GetMultipleAccountsWithOpts wraps the RPC call with rate limiting
func (c *Client) GetMultipleAccountsWithOpts(ctx context.Context, accounts []solana.PublicKey, opts *rpc.GetMultipleAccountsOpts) (*rpc.GetMultipleAccountsResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetMultipleAccountsWithOpts(ctx, accounts, opts)
}

// GetProgramAccounts wraps the RPC call with rate limiting
func (c *Client) GetProgramAccounts(ctx context.Context, programID solana.PublicKey) (rpc.GetProgramAccountsResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetProgramAccounts(ctx, programID)
}

// GetProgramAccountsWithOpts wraps the RPC call with rate limiting
func (c *Client) GetProgramAccountsWithOpts(ctx context.Context, programID solana.PublicKey, opts *rpc.GetProgramAccountsOpts) (rpc.GetProgramAccountsResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetProgramAccountsWithOpts(ctx, programID, opts)
}

// GetTokenAccountsByOwner wraps the RPC call with rate limiting
func (c *Client) GetTokenAccountsByOwner(ctx context.Context, owner solana.PublicKey, config *rpc.GetTokenAccountsConfig, opts *rpc.GetTokenAccountsOpts) (*rpc.GetTokenAccountsResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetTokenAccountsByOwner(ctx, owner, config, opts)
}

// GetTokenAccountBalance wraps the RPC call with rate limiting
func (c *Client) GetTokenAccountBalance(ctx context.Context, account solana.PublicKey, commitment rpc.CommitmentType) (*rpc.GetTokenAccountBalanceResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetTokenAccountBalance(ctx, account, commitment)
}

// GetLatestBlockhash wraps the RPC call with rate limiting
func (c *Client) GetLatestBlockhash(ctx context.Context, commitment rpc.CommitmentType) (*rpc.GetLatestBlockhashResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetLatestBlockhash(ctx, commitment)
}

// SimulateTransaction wraps the RPC call with rate limiting
func (c *Client) SimulateTransaction(ctx context.Context, tx *solana.Transaction) (*rpc.SimulateTransactionResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.SimulateTransaction(ctx, tx)
}

// SendTransaction wraps the RPC call with rate limiting
func (c *Client) SendTransaction(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return solana.Signature{}, err
	}
	return c.rpcClient.SendTransaction(ctx, tx)
}

// SendTransactionWithOpts wraps the RPC call with rate limiting
func (c *Client) SendTransactionWithOpts(ctx context.Context, tx *solana.Transaction, opts rpc.TransactionOpts) (solana.Signature, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return solana.Signature{}, err
	}
	return c.rpcClient.SendTransactionWithOpts(ctx, tx, opts)
}

// GetBalance wraps the RPC call with rate limiting
func (c *Client) GetBalance(ctx context.Context, account solana.PublicKey, commitment rpc.CommitmentType) (*rpc.GetBalanceResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetBalance(ctx, account, commitment)
}
