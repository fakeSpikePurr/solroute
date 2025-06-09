package main

import (
	"context"
	"log"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/yimingwow/solroute/pkg/protocol"
	"github.com/yimingwow/solroute/pkg/router"
	"github.com/yimingwow/solroute/pkg/sol"
)

const (
	// RPC endpoints
	mainnetRPC = ""

	// Token addresses
	usdcTokenAddr = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"

	// Swap parameters
	defaultAmountIn = 1000000000 // 1 sol (9 decimals)
	slippageBps     = 100        // 1% slippage
)

func main() {
	// TODO: Initialize private key from environment or config file
	var privateKey solana.PrivateKey

	ctx := context.Background()
	solClient, err := sol.NewClient(ctx, mainnetRPC)
	if err != nil {
		log.Fatalf("Failed to create solana client: %v", err)
	}
	defer solClient.Close()

	router := router.NewSimpleRouter(
		protocol.NewPumpAmm(solClient),
		protocol.NewRaydiumAmm(solClient),
		protocol.NewRaydiumClmm(solClient),
		protocol.NewRaydiumCpmm(solClient),
	)

	// Query available pools
	pools, err := router.QueryAllPools(ctx, usdcTokenAddr, sol.WSOL.String())
	if err != nil {
		log.Fatalf("Failed to query all pools: %v", err)
	}
	for _, pool := range pools {
		log.Printf("Found pool: %v", pool.GetID())
	}

	// Find best pool for the swap
	amountIn := math.NewInt(defaultAmountIn)
	bestPool, amountOut, err := router.GetBestPool(ctx, solClient.RpcClient, sol.WSOL.String(), usdcTokenAddr, amountIn)
	if err != nil {
		log.Fatalf("Failed to get best pool: %v", err)
	}
	log.Printf("Selected best pool: %v", bestPool.GetID())
	log.Printf("Expected output amount: %v", amountOut)

	// Calculate minimum output amount with slippage
	minAmountOut := amountOut.Mul(math.NewInt(10000 - slippageBps)).Quo(math.NewInt(10000))

	// Build swap instructions
	instructions, err := bestPool.BuildSwapInstructions(ctx, solClient.RpcClient,
		privateKey.PublicKey(), usdcTokenAddr, amountIn, minAmountOut)
	if err != nil {
		log.Fatalf("Failed to build swap instructions: %v", err)
	}
	log.Printf("Generated swap instructions: %v", instructions)

	// Prepare transaction
	signers := []solana.PrivateKey{privateKey}
	res, err := solClient.RpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("Failed to get blockhash: %v", err)
	}

	// Send transaction
	sig, err := solClient.SendTx(ctx, res.Value.Blockhash, signers, instructions, true)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}
	log.Printf("Transaction successful: https://solscan.io/tx/%v", sig)
}
