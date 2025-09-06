package sol

import (
	"context"
	"fmt"
	"log"

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
func (c *Client) SendTx(ctx context.Context, signers []solana.PrivateKey, insts []solana.Instruction, isSimulate bool) (solana.Signature, error) {
	res, err := c.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to get blockhash: %v", err)
	}

	tx, err := signTransaction(res.Value.Blockhash, signers, insts...)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if isSimulate {
		if _, err := c.SimulateTransaction(ctx, tx); err != nil {
			return solana.Signature{}, fmt.Errorf("failed to simulate transaction: %w", err)
		}
		// Return empty signature for simulation
		return solana.Signature{}, nil
	}

	// Send transaction with optimized options
	sig, err := c.SendTransactionWithOpts(
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
