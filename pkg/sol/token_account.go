package sol

import (
	"context"
	"log"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/rpc"
)

func (t *Client) SelectOrCreateSPLTokenAccount(ctx context.Context, privateKey solana.PrivateKey, tokenMint solana.PublicKey) (solana.PublicKey, error) {
	user := privateKey.PublicKey()
	acc, err := t.GetTokenAccountsByOwner(ctx, user,
		&rpc.GetTokenAccountsConfig{Mint: tokenMint.ToPointer()},
		&rpc.GetTokenAccountsOpts{
			Encoding: "jsonParsed",
		},
	)
	if err != nil {
		log.Printf("GetTokenAccountsByOwner err: %v", err)
		return solana.PublicKey{}, err
	}
	if len(acc.Value) > 0 {
		return acc.Value[0].Pubkey, nil
	}

	// Find ATA address (this will always return a valid PDA)
	ataAddress, _, err := solana.FindAssociatedTokenAddress(user, tokenMint)
	if err != nil {
		log.Printf("FindAssociatedTokenAddress err: %v", err)
		return solana.PublicKey{}, err
	}
	instructions := make([]solana.Instruction, 0)
	createAtaInst, err := associatedtokenaccount.NewCreateInstruction(
		user,
		user,
		tokenMint,
	).ValidateAndBuild()
	if err != nil {
		return solana.PublicKey{}, err
	}
	instructions = append(instructions, createAtaInst)

	if len(instructions) == 0 {
		return ataAddress, nil
	} else {
		signers := []solana.PrivateKey{privateKey}
		tx, err := t.SignTransaction(ctx, signers, instructions...)
		if err != nil {
			log.Printf("Failed to sign transaction: %v", err)
			return solana.PublicKey{}, err
		}
		_, err = t.SendTx(ctx, tx)
		if err != nil {
			log.Printf("Failed to send transaction: %v", err)
			return solana.PublicKey{}, err
		}
		return ataAddress, nil
	}
}
