package pkg

import (
	"context"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type Pool interface {
	GetID() string
	GetTokens() (baseMint, quoteMint string)
	GetType() PoolType
	GetQuote(ctx context.Context, inputMint string, inputAmount math.Int) (math.Int, error)
	BuildSwapInstructions(
		ctx context.Context,
		solClient *rpc.Client,
		user solana.PublicKey,
		inputMint string,
		inputAmount math.Int,
		minOut math.Int,
	) ([]solana.Instruction, error)
}

type Protocol interface {
	FetchPoolsByPair(ctx context.Context, baseMint, quoteMint string) ([]Pool, error)
	FetchPoolByID(ctx context.Context, poolID string) (Pool, error)
}
