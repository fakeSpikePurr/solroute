package main

import (
	"context"
	"flag"
	"log"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"github.com/yimingWOW/solroute/config"
	"github.com/yimingWOW/solroute/pkg/protocol"
	"github.com/yimingWOW/solroute/pkg/router"
	"github.com/yimingWOW/solroute/pkg/sol"
	"github.com/zeromicro/go-zero/core/conf"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

const (
	// Token addresses
	usdcTokenAddr = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"

	// Swap parameters
	defaultAmountIn = 10000000 // 0.01 sol (9 decimals)
	solDecimal      = float64(1e9)
	slippageBps     = 100 // 1% slippage
)

func main() {
	log.Printf("🚀🚀🚀parpering to earn...")

	var config config.Config
	configFile := flag.String("f", "./config/config.json", "the config file")
	conf.MustLoad(*configFile, &config)

	// TODO: Initialize private key from environment or config file
	privateKey := solana.MustPrivateKeyFromBase58(config.PrivateKey)
	log.Printf("😈get your public key: %v", privateKey.PublicKey())

	ctx := context.Background()
	solClient, err := sol.NewClient(ctx, config.RPC, config.WSRPC, 20) // 50 requests per second
	if err != nil {
		log.Fatalf("Failed to create solana client: %v", err)
	}
	defer solClient.Close()

	// check balance first
	balance, err := solClient.GetUserTokenBalance(ctx, privateKey.PublicKey(), sol.WSOL)
	if err != nil {
		log.Fatalf("Failed to get user token balance: %v", err)
	}
	log.Printf("😈You have %v wsol", balance)
	if balance < defaultAmountIn {
		log.Printf("🧐You don't have enough wsol, covering %f wsol...", float64(defaultAmountIn)/solDecimal)
		err = solClient.CoverWsol(ctx, privateKey, defaultAmountIn)
		if err != nil {
			log.Fatalf("Failed to cover wsol: %v", err)
		}
	}
	tokenAccount, err := solClient.SelectOrCreateSPLTokenAccount(ctx, privateKey, solana.MustPublicKeyFromBase58(usdcTokenAddr))
	if err != nil {
		log.Fatalf("Failed to get user token balance: %v", err)
	}
	log.Printf("😈Your USDC token account: %v", tokenAccount.String())

	router := router.NewSimpleRouter(
		protocol.NewPumpAmm(solClient),
		protocol.NewRaydiumAmm(solClient),
		protocol.NewRaydiumClmm(solClient),
		protocol.NewRaydiumCpmm(solClient),
		protocol.NewMeteoraDlmm(solClient),
	)

	// Query available pools
	log.Printf("⌛️Querying available pools...")
	err = router.QueryAllPools(ctx, usdcTokenAddr, sol.WSOL.String())
	if err != nil {
		log.Fatalf("Failed to query all pools: %v", err)
	}
	log.Printf("👌Found %d pools", len(router.Pools))

	// Find best pool for the swap
	amountIn := math.NewInt(defaultAmountIn)
	bestPool, amountOut, err := router.GetBestPool(ctx, solClient, sol.WSOL.String(), amountIn)
	if err != nil {
		log.Fatalf("Failed to get best pool: %v", err)
	}
	log.Printf("Selected best pool: %v", bestPool.GetID())
	log.Printf("Expected output amount: %v", amountOut)

	// Calculate minimum output amount with slippage
	minAmountOut := amountOut.Mul(math.NewInt(10000 - slippageBps)).Quo(math.NewInt(10000))

	// Build swap instructions
	instructions, err := bestPool.BuildSwapInstructions(ctx, solClient,
		privateKey.PublicKey(), usdcTokenAddr, amountIn, minAmountOut)
	if err != nil {
		log.Fatalf("Failed to build swap instructions: %v", err)
	}
	log.Printf("Generated swap instructions: %v", instructions)

	// Send transaction
	isSimulate := true
	sig, err := solClient.SendTx(ctx, []solana.PrivateKey{privateKey}, instructions, isSimulate)
	if err != nil {
		log.Fatalf("Failed to SendTx: %v", err)
	}
	log.Printf("Transaction successful: https://solscan.io/tx/%v", sig)
}
