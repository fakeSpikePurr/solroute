package main

import (
	"context"
	"log"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"github.com/yimingwow/solroute/pkg/protocol"
	"github.com/yimingwow/solroute/pkg/router"
	"github.com/yimingwow/solroute/pkg/sol"
)

var (
	privateKey = ""
	rpc        = ""
	jitoRpc    = ""
	// Token addresses
	inTokenAddr  = sol.WSOL
	outTokenAddr = solana.MustPublicKeyFromBase58("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")

	// Swap parameters
	defaultAmountIn = int64(10000000) // 0.01 sol (9 decimals)
	solDecimal      = float64(1e9)
	slippageBps     = 100 // 1% slippage
	useJito         = false
	isSimulate      = true
)

func main() {
	log.Printf("🚀🚀🚀parpering to earn...")

	privateKey := solana.MustPrivateKeyFromBase58(privateKey)
	log.Printf("😈get your public key: %v", privateKey.PublicKey())

	ctx := context.Background()
	solClient, err := sol.NewClient(ctx, rpc, jitoRpc, 20) // 50 requests per second
	if err != nil {
		log.Fatalf("Failed to create solana client: %v", err)
	}

	// check balance first
	inTokenAccount, balance, err := solClient.GetUserTokenBalance(ctx, privateKey.PublicKey(), inTokenAddr)
	if err != nil && err.Error() != "no token account found" {
		log.Fatalf("Failed to get user token balance: %v", err)
	}
	log.Printf("😈You have %v wsol", balance)
	if err != nil || balance < uint64(defaultAmountIn) {
		log.Printf("🧐You don't have enough wsol, covering %f wsol...", float64(defaultAmountIn)/solDecimal)
		err = solClient.CoverWsol(ctx, privateKey, defaultAmountIn)
		if err != nil {
			log.Fatalf("Failed to cover wsol: %v", err)
		}
	}
	outTokenAccount, err := solClient.SelectOrCreateSPLTokenAccount(ctx, privateKey, outTokenAddr)
	if err != nil {
		log.Fatalf("Failed to get user token balance: %v", err)
	}
	log.Printf("😈Your token account: %v", outTokenAccount.String())

	router := router.NewSimpleRouter(
		protocol.NewPumpAmm(solClient),
		protocol.NewRaydiumAmm(solClient),
		protocol.NewRaydiumClmm(solClient),
		protocol.NewRaydiumCpmm(solClient),
		protocol.NewMeteoraDlmm(solClient),
	)

	// Query available pools
	log.Printf("⌛️Querying available pools...")
	err = router.QueryAllPools(ctx, inTokenAddr.String(), outTokenAddr.String())
	if err != nil {
		log.Fatalf("Failed to query all pools: %v", err)
	}
	log.Printf("👌Found %d pools", len(router.Pools))

	signers := []solana.PrivateKey{}
	instructions := make([]solana.Instruction, 0)

	amountIn := math.NewInt(defaultAmountIn)
	bestPool, amountOut, err := router.GetBestPool(ctx, solClient, inTokenAddr.String(), amountIn)
	if err != nil {
		log.Fatalf("Failed to get best pool: %v", err)
	}
	log.Printf("Selected best pool: %v, amountOut: %v", bestPool.GetID(), amountOut)

	minAmountOut := amountOut.Mul(math.NewInt(int64(10000 - slippageBps))).Quo(math.NewInt(10000))
	instructionsBuy, err := bestPool.BuildSwapInstructions(ctx, solClient,
		privateKey.PublicKey(), inTokenAddr.String(), amountIn, minAmountOut, inTokenAccount, outTokenAccount)
	if err != nil {
		log.Fatalf("Failed to build swap instructions: %v", err)
	}
	signers = append(signers, privateKey)
	instructions = append(instructions, instructionsBuy...)

	tx, err := solClient.SignTransaction(ctx, signers, instructions...)
	if err != nil {
		log.Fatalf("Failed to SendTx: %v", err)
	}

	if isSimulate {
		if _, err := solClient.SimulateTransaction(ctx, tx); err != nil {
			log.Fatalf("Failed to simulate transaction: %v", err)
		}
	}
	if useJito {
		_, err = solClient.SendTxWithJito(ctx, 1000000, signers, tx)
		if err != nil {
			log.Fatalf("Failed to SendTxWithJito: %v", err)
		}
	} else {
		sig, err := solClient.SendTx(ctx, tx)
		if err != nil {
			log.Fatalf("Failed to SendTx: %v", err)
		}
		log.Printf("Transaction successful: https://solscan.io/tx/%v", sig)
	}
}
