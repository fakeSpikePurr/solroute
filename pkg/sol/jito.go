package sol

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	jitorpc "github.com/jito-labs/jito-go-rpc"
)

type JitoClient struct {
	rpcClient  *jitorpc.JitoJsonRpcClient
	tipAccount solana.PublicKey
}

// Jito endpoint refer to: https://docs.jito.wtf/lowlatencytxnsend/
func NewJitoClient(ctx context.Context, endpoint string) (*JitoClient, error) {
	rpcClient := jitorpc.NewJitoJsonRpcClient(endpoint, "")
	tipAccount, err := rpcClient.GetRandomTipAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to get random tip account: %v", err)
	}
	tipAccountPublicKey, err := solana.PublicKeyFromBase58(tipAccount.Address)
	return &JitoClient{
		rpcClient:  rpcClient,
		tipAccount: tipAccountPublicKey,
	}, nil
}

func createTipTransaction(privateKey solana.PrivateKey, amount uint64, recentBlockhash solana.Hash, tipAddress string) (*solana.Transaction, error) {
	tipAccount, err := solana.PublicKeyFromBase58(tipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tip account: %v", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				amount,
				privateKey.PublicKey(),
				tipAccount,
			).Build(),
		},
		recentBlockhash,
		solana.TransactionPayer(privateKey.PublicKey()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tip transaction: %v", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if privateKey.PublicKey().Equals(key) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign tip transaction: %v", err)
	}

	return tx, nil
}

func encodeTransaction(tx *solana.Transaction) string {
	serializedTx, err := tx.MarshalBinary()
	if err != nil {
		log.Fatalf("Failed to serialize transaction: %v", err)
	}
	return base64.StdEncoding.EncodeToString(serializedTx)
}

func (c *JitoClient) CheckBundleStatus(bundleId string) {
	maxAttempts := 5
	pollInterval := 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(pollInterval)

		statusResponse, err := c.rpcClient.GetBundleStatuses([]string{bundleId})
		if err != nil {
			log.Printf("Attempt %d: Failed to get bundle status: %v", attempt, err)
			continue
		}

		if len(statusResponse.Value) == 0 {
			log.Printf("Attempt %d: No bundle status available", attempt)
			continue
		}

		bundleStatus := statusResponse.Value[0]
		log.Printf("Attempt %d: Bundle status: %s", attempt, bundleStatus.ConfirmationStatus)

		switch bundleStatus.ConfirmationStatus {
		case "processed":
			fmt.Println("Bundle has been processed by the cluster. Continuing to poll...")
		case "confirmed":
			fmt.Println("Bundle has been confirmed by the cluster. Continuing to poll...")
		case "finalized":
			fmt.Printf("Bundle has been finalized by the cluster in slot %d.\n", bundleStatus.Slot)
			if bundleStatus.Err.Ok == nil {
				fmt.Println("Bundle executed successfully.")
				fmt.Println("Transaction URLs:")
				for _, txID := range bundleStatus.Transactions {
					solscanURL := fmt.Sprintf("https://solscan.io/tx/%s", txID)
					fmt.Printf("- %s\n", solscanURL)
				}
			} else {
				fmt.Printf("Bundle execution failed with error: %v\n", bundleStatus.Err.Ok)
			}
			return
		default:
			fmt.Printf("Unexpected status: %s. Please check the bundle manually.\n", bundleStatus.ConfirmationStatus)
			return
		}
	}

	log.Printf("Maximum polling attempts reached. Final status unknown.")
}
