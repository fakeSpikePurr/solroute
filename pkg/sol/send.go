package sol

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func (c *Client) SendTx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
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

func (c *Client) SendTxWithJito(ctx context.Context, jitoTipAmount uint64, signers []solana.PrivateKey, mainTx *solana.Transaction) (string, error) {

	res, err := c.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to get blockhash: %v", err)
	}

	tipTx, err := createTipTransaction(signers[0], jitoTipAmount, res.Value.Blockhash, c.jitoClient.tipAccount.String())
	if err != nil {
		log.Fatalf("Failed to create tip transaction: %v", err)
	}

	bundleRequest := [][]string{{
		encodeTransaction(mainTx),
		encodeTransaction(tipTx),
	}}

	bundleIdRaw, err := c.jitoClient.rpcClient.SendBundle(bundleRequest)
	if err != nil {
		log.Fatalf("Failed to send bundle: %v", err)
	}
	var bundleId string
	if err := json.Unmarshal(bundleIdRaw, &bundleId); err != nil {
		log.Fatalf("Failed to unmarshal bundle ID: %v", err)
	}

	fmt.Printf("Bundle sent successfully. Bundle ID: %s\n", bundleId)
	c.jitoClient.CheckBundleStatus(bundleId)

	return bundleId, nil
}
