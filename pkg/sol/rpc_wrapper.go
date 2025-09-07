package sol

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// RPC wrapper methods with rate limiting

// GetAccountInfoWithOpts wraps the RPC call with rate limiting
func (c *Client) GetAccountInfoWithOpts(ctx context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	opts := &rpc.GetAccountInfoOpts{
		Commitment: rpc.CommitmentProcessed,
	}
	return c.rpcClient.GetAccountInfoWithOpts(ctx, account, opts)
}

// GetMultipleAccountsWithOpts wraps the RPC call with rate limiting
func (c *Client) GetMultipleAccountsWithOpts(ctx context.Context, accounts []solana.PublicKey) (*rpc.GetMultipleAccountsResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	opts := &rpc.GetMultipleAccountsOpts{
		Commitment: rpc.CommitmentProcessed,
	}
	return c.rpcClient.GetMultipleAccountsWithOpts(ctx, accounts, opts)
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

// GetBalance wraps the RPC call with rate limiting
func (c *Client) GetBalance(ctx context.Context, account solana.PublicKey, commitment rpc.CommitmentType) (*rpc.GetBalanceResult, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}
	return c.rpcClient.GetBalance(ctx, account, commitment)
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

// SendTransactionWithOpts wraps the RPC call with rate limiting
func (c *Client) SendTransactionWithOpts(ctx context.Context, tx *solana.Transaction, opts rpc.TransactionOpts) (solana.Signature, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return solana.Signature{}, err
	}
	return c.rpcClient.SendTransactionWithOpts(ctx, tx, opts)
}
