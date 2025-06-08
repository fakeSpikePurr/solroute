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

func main() {
	var privateKey solana.PrivateKey

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

	instructions, err := bestPool.BuildSwapInstructions(ctx, solClient.RpcClient,
		privateKey.PublicKey(), tokenA, math.NewInt(1000000000), math.NewInt(0))
	if err != nil {
		log.Fatalf("Failed to build swap instructions: %v", err)
	}
	log.Printf("Instructions: %v", instructions)

	signers := make([]solana.PrivateKey, 0)
	signers = append(signers, privateKey)

	res, err := solClient.RpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		log.Printf("Failed to get blockhash: %v\n", err)
		return
	}

	sig, err := solClient.SendTx(ctx, res.Value.Blockhash, signers, instructions, true)
	if err != nil {
		log.Printf("Failed to send tx: %v\n", err)
		return
	}
	log.Printf("---Transaction successfully: https://solscan.io/tx/%v\n", sig)

}
