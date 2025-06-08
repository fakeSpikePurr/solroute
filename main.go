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

func main() {
	ctx := context.Background()
	solClient, err := sol.NewClient(ctx, "https://api.devnet.solana.com", "https://jito-rpc.mainnet-beta.solana.com")
	if err != nil {
		log.Fatalf("Failed to create solana client: %v", err)
	}

	router := router.NewSimpleRouter(
		protocol.NewPumpAmm(solClient),
		protocol.NewRaydiumAmm(solClient),
	)

	tokenA := "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" // usdc
	tokenB := sol.WSOL.String()

	pools, err := router.QueryAllPools(ctx, tokenA, tokenB)
	if err != nil {
		log.Fatalf("Failed to query all pools: %v", err)
	}
	for _, pool := range pools {
		log.Printf("Pool: %v", pool.GetID())
	}

	bestPool, amountOut, err := router.GetBestPool(ctx, tokenA, tokenB, math.NewInt(1000000000))
	if err != nil {
		log.Fatalf("Failed to get best pool: %v", err)
	}
	log.Printf("Best pool: %v", bestPool.GetID())
	log.Printf("Amount out: %v", amountOut)

	user := solana.MustPublicKeyFromBase58("568998654321")
	instructions, err := bestPool.BuildSwapInstructions(ctx, solClient.RpcClient,
		user, tokenA, math.NewInt(1000000000), math.NewInt(0))
	if err != nil {
		log.Fatalf("Failed to build swap instructions: %v", err)
	}
	log.Printf("Instructions: %v", instructions)
}
