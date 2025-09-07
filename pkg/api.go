package pkg

import (
	"context"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"github.com/yimingwow/solroute/pkg/sol"
)

// ProtocolName represents the string name of AMM protocol
type ProtocolName string

const (
	ProtocolNameRaydiumAmm  ProtocolName = "raydium_amm"
	ProtocolNameRaydiumClmm ProtocolName = "raydium_clmm"
	ProtocolNameRaydiumCpmm ProtocolName = "raydium_cpmm"
	ProtocolNameMeteoraDlmm ProtocolName = "meteora_dlmm"
	ProtocolNamePumpAmm     ProtocolName = "pump_amm"
)

type Pool interface {
	ProtocolName() ProtocolName
	GetProgramID() solana.PublicKey
	GetID() string
	GetTokens() (baseMint, quoteMint string)
	Quote(ctx context.Context, solClient *sol.Client, inputMint string, inputAmount math.Int) (math.Int, error)
	BuildSwapInstructions(
		ctx context.Context,
		solClient *sol.Client,
		user solana.PublicKey,
		inputMint string,
		inputAmount math.Int,
		minOut math.Int,
		userBaseAccount solana.PublicKey,
		userQuoteAccount solana.PublicKey,
	) ([]solana.Instruction, error)
}

type Protocol interface {
	ProtocolName() ProtocolName
	FetchPoolsByPair(ctx context.Context, baseMint, quoteMint string) ([]Pool, error)
	FetchPoolByID(ctx context.Context, poolID string) (Pool, error)
}
