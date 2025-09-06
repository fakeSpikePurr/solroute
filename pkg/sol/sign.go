package sol

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func (c *Client) SignTransaction(ctx context.Context, signers []solana.PrivateKey, instrs ...solana.Instruction) (*solana.Transaction, error) {

	if len(signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}

	res, err := c.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to get blockhash: %v", err)
	}

	// Create new transaction with all instructions
	tx, err := solana.NewTransaction(
		instrs,
		res.Value.Blockhash,
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
