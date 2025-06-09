package sol

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// signTransaction creates and signs a new transaction with the given instructions
func signTransaction(blockhash solana.Hash, signers []solana.PrivateKey, instrs ...solana.Instruction) (*solana.Transaction, error) {
	if len(signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}

	// Create new transaction with all instructions
	tx, err := solana.NewTransaction(
		instrs,
		blockhash,
		solana.TransactionPayer(signers[0].PublicKey()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign the transaction with all provided signers
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			for _, payer := range signers {
				if payer.PublicKey().Equals(key) {
					return &payer
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	return tx, nil
}

// SendTx sends or simulates a transaction based on the isSimulate flag
func (c *Client) SendTx(ctx context.Context, blockhash solana.Hash, signers []solana.PrivateKey, insts []solana.Instruction, isSimulate bool) (solana.Signature, error) {
	tx, err := signTransaction(blockhash, signers, insts...)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if isSimulate {
		if _, err := c.RpcClient.SimulateTransaction(ctx, tx); err != nil {
			return solana.Signature{}, fmt.Errorf("failed to simulate transaction: %w", err)
		}
		// Return empty signature for simulation
		return solana.Signature{}, nil
	}

	// Send transaction with optimized options
	sig, err := c.RpcClient.SendTransactionWithOpts(
		ctx, tx,
		rpc.TransactionOpts{
			SkipPreflight:       true,
			PreflightCommitment: rpc.CommitmentProcessed,
		},
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send transaction: %w", err)
	}
	return sig, nil
}
