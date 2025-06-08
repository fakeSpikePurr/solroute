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
	) ([]solana.Instruction, []solana.PrivateKey, error)
}

type Protocol interface {
	Name() string
	FetchPoolsByPair(baseMint, quoteMint string) ([]Pool, error)
	FetchPoolByID(poolID string) (Pool, error)
}
